package runner

import (
	"reflect"
	"testing"

	"kronize/internal/model"
)

func TestBuildEnvAlwaysUnbuffered(t *testing.T) {
	job := &model.Job{EnvVars: `{"A":"b"}`}
	got := buildEnv(job)
	want := []string{"A=b", "PYTHONUNBUFFERED=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildEnv() = %#v, want %#v", got, want)
	}
}

func TestBuildEnvNoJobEnvVars(t *testing.T) {
	job := &model.Job{EnvVars: "{}"}
	got := buildEnv(job)
	want := []string{"PYTHONUNBUFFERED=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildEnv() = %#v, want %#v", got, want)
	}
}

func TestBuildMountsRejectsUnsafeHostPaths(t *testing.T) {
	for _, raw := range []string{
		`[{"host":"/","container":"/hostroot","read_only":false}]`,
		`[{"host":"/var/run/docker.sock","container":"/sock","read_only":false}]`,
		`[{"host":"/etc","container":"/hostetc","read_only":true}]`,
	} {
		if _, err := buildMounts(&model.Job{HostMappings: raw}); err == nil {
			t.Fatalf("buildMounts(%s) should fail, got nil", raw)
		}
	}

	mounts, err := buildMounts(&model.Job{HostMappings: `[{"host":"/mnt/data","container":"/input","read_only":true}]`})
	if err != nil {
		t.Fatalf("buildMounts valid failed: %v", err)
	}
	if len(mounts) != 1 || mounts[0].Source != "/mnt/data" || mounts[0].Target != "/input" || !mounts[0].ReadOnly {
		t.Fatalf("unexpected mounts: %+v", mounts)
	}
}

func TestBuildContainerConfigsHardened(t *testing.T) {
	cfg, hostCfg := buildContainerConfigs("kronize/python-runner", []string{"PYTHONUNBUFFERED=1"}, nil)
	if cfg.User != "65534:65534" {
		t.Fatalf("User = %q, want 65534:65534", cfg.User)
	}
	if !hostCfg.ReadonlyRootfs {
		t.Fatalf("ReadonlyRootfs = false, want true")
	}
	if !reflect.DeepEqual([]string(hostCfg.CapDrop), []string{"ALL"}) {
		t.Fatalf("CapDrop = %#v, want [ALL]", hostCfg.CapDrop)
	}
	if !reflect.DeepEqual(hostCfg.SecurityOpt, []string{"no-new-privileges:true"}) {
		t.Fatalf("SecurityOpt = %#v, want [no-new-privileges:true]", hostCfg.SecurityOpt)
	}
	if _, ok := hostCfg.Tmpfs["/tmp"]; !ok {
		t.Fatalf("expected /tmp tmpfs mount in HostConfig")
	}
}
