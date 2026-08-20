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
