package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// connectionInfo mirrors the body install.ps1 expects at
// $env:CATTLE_AGENT_VAR_DIR/rancher2_connection_info.json, and what
// pkg/config.ConnectionInfo (github.com/rancher/system-agent/pkg/config) decodes.
type connectionInfo struct {
	KubeConfig string `json:"kubeConfig"`
	Namespace  string `json:"namespace"`
	SecretName string `json:"secretName"`
}

func bootstrapCommand() *cli.Command {
	return &cli.Command{
		Name: "bootstrap",
		Usage: "idempotently create the namespace, ServiceAccount, Role, and RoleBinding; reset the plan Secret to empty;" +
			" and print the agent kubeconfig plus connection info JSON",
		Action: func(cCtx *cli.Context) error {
			clientset, err := newClientset(cCtx)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if err := ensureNamespace(ctx, clientset); err != nil {
				return err
			}
			// Reset rather than create-if-absent: bootstrap must not leave a prior run's
			// plan/plan-state/annotations in place, whether that prior run finished cleanly or
			// was interrupted mid-spec (e.g. left canceled or paused).
			if err := resetPlanSecret(ctx, clientset); err != nil {
				return err
			}
			if err := ensureServiceAccount(ctx, clientset); err != nil {
				return err
			}
			if err := ensureRoleAndBinding(ctx, clientset); err != nil {
				return err
			}

			token, err := mintToken(ctx, clientset)
			if err != nil {
				return err
			}

			kubeconfigPath := cCtx.String("kubeconfig")
			restCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			if err != nil {
				return fmt.Errorf("building rest config from %s: %w", kubeconfigPath, err)
			}

			agentKubeconfig, err := buildAgentKubeconfig(restCfg, token)
			if err != nil {
				return err
			}

			connInfo := connectionInfo{
				KubeConfig: agentKubeconfig,
				Namespace:  namespace,
				SecretName: secretName,
			}
			connInfoJSON, err := json.Marshal(connInfo)
			if err != nil {
				return fmt.Errorf("marshalling connection info: %w", err)
			}

			out := struct {
				AgentKubeConfig    string `json:"agentKubeConfig"`
				ConnectionInfoJSON string `json:"connectionInfoJson"`
			}{
				AgentKubeConfig:    agentKubeconfig,
				ConnectionInfoJSON: string(connInfoJSON),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
}

func ensureNamespace(ctx context.Context, clientset *kubernetes.Clientset) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	_, err := clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating namespace %s: %w", namespace, err)
	}
	return nil
}

func ensureServiceAccount(ctx context.Context, clientset *kubernetes.Clientset) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
		},
	}
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating service account %s/%s: %w", namespace, saName, err)
	}
	return nil
}

func ensureRoleAndBinding(ctx context.Context, clientset *kubernetes.Clientset) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{secretName},
				Verbs:         []string{"get", "list", "watch", "update"},
			},
		},
	}
	if _, err := clientset.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating role %s/%s: %w", namespace, roleName, err)
	}

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     roleName,
		},
	}
	if _, err := clientset.RbacV1().RoleBindings(namespace).Create(ctx, binding, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating role binding %s/%s: %w", namespace, roleName, err)
	}
	return nil
}

func mintToken(ctx context.Context, clientset *kubernetes.Clientset) (string, error) {
	expirationSeconds := int64((24 * time.Hour).Seconds())
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}
	result, err := clientset.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saName, tr, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("minting token for %s/%s: %w", namespace, saName, err)
	}
	return result.Status.Token, nil
}

// buildAgentKubeconfig builds a minimal kubeconfig for the scoped ServiceAccount, reusing the
// admin kubeconfig's server address and CA material and substituting the minted token.
func buildAgentKubeconfig(restCfg *rest.Config, token string) (string, error) {
	const clusterName = "wins-plan-e2e-cluster"
	const userName = "wins-plan-e2e-user"
	const contextName = "wins-plan-e2e-context"

	cluster := &clientcmdapi.Cluster{
		Server:                   restCfg.Host,
		InsecureSkipTLSVerify:    restCfg.Insecure,
		CertificateAuthority:     restCfg.CAFile,
		CertificateAuthorityData: restCfg.CAData,
	}

	kc := clientcmdapi.Config{
		Clusters:  map[string]*clientcmdapi.Cluster{clusterName: cluster},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{userName: {Token: token}},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {Cluster: clusterName, AuthInfo: userName, Namespace: namespace},
		},
		CurrentContext: contextName,
	}

	raw, err := clientcmd.Write(kc)
	if err != nil {
		return "", fmt.Errorf("writing agent kubeconfig: %w", err)
	}
	return string(raw), nil
}
