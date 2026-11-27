package model

import "testing"

func TestValidateRunnerImageRef(t *testing.T) {
	valid := []string{
		"python:3.12-slim",
		"kronize/python-runner",
		"kronize/python-runner:v1.0.8",
		"cr.prolifel.com/kronize/python-runner:python-runner-v1.0.8",
		"ghcr.io/org/repo:latest",
		"registry.example.com:443/org/repo:v1",
	}
	for _, ref := range valid {
		if err := ValidateRunnerImageRef(ref); err != nil {
			t.Fatalf("ValidateRunnerImageRef(%q) unexpected error: %v", ref, err)
		}
	}

	invalid := []string{
		"",
		"   ",
		"127.0.0.1:8080/x:v1",
		"127.0.0.1/x:v1",
		"172.19.0.1:2375/x:v1",
		"10.0.0.1:443/x:v1",
		"169.254.169.254/latest/meta-data:v1",
		"[::1]:2376/x:v1",
		"host.docker.internal:8080/x:v1",
		"host.docker.internal/x:v1",
		"localhost:5000/x:v1",
		"localhost/x:v1",
		"mybox.local/x:v1",
		"registry.example.com:8443/x:v1",
		"https://cr.prolifel.com/x:v1",
		"cr.prolifel.com/x:v1?foo=bar",
	}
	for _, ref := range invalid {
		if err := ValidateRunnerImageRef(ref); err == nil {
			t.Fatalf("ValidateRunnerImageRef(%q) expected error, got nil", ref)
		}
	}

	t.Setenv("ALLOWED_RUNNER_REGISTRIES", "cr.prolifel.com")
	if err := ValidateRunnerImageRef("cr.prolifel.com/kronize/python-runner:v1"); err != nil {
		t.Fatalf("allowlisted registry failed: %v", err)
	}
	if err := ValidateRunnerImageRef("ghcr.io/org/repo:v1"); err == nil {
		t.Fatalf("non-allowlisted registry should fail when ALLOWED_RUNNER_REGISTRIES is set")
	}
}
