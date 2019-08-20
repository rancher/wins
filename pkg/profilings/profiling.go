package profilings

import (
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/pkg/errors"
)

func Init(profileName string, profileOutput string) error {
	switch profileName {
	case "none":
		return nil
	case "cpu":
		f, err := os.Create(profileOutput)
		if err != nil {
			return err
		}
		return pprof.StartCPUProfile(f)
	case "block":
		runtime.SetBlockProfileRate(1)
		return nil
	case "mutex":
		runtime.SetMutexProfileFraction(1)
		return nil
	default:
		if profile := pprof.Lookup(profileName); profile == nil {
			return errors.Errorf("unknown profile %q", profileName)
		}
	}

	return nil
}

func Flush(profileName string, profileOutput string) error {
	switch profileName {
	case "none":
		return nil
	case "cpu":
		pprof.StopCPUProfile()
	case "heap":
		runtime.GC()
		fallthrough
	default:
		profile := pprof.Lookup(profileName)
		if profile == nil {
			return nil
		}
		f, err := os.Create(profileOutput)
		if err != nil {
			return err
		}
		profile.WriteTo(f, 0)
	}

	return nil
}
