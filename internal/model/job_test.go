package model

import "testing"

func TestParseAndValidateHostMappings(t *testing.T) {
	// Empty / default values are allowed.
	for _, raw := range []string{"", "[]", "  []  "} {
		got, err := ParseAndValidateHostMappings(raw)
		if err != nil {
			t.Fatalf("ParseAndValidateHostMappings(%q) unexpected error: %v", raw, err)
		}
		if len(got) != 0 {
			t.Fatalf("ParseAndValidateHostMappings(%q) = %v, want empty", raw, got)
		}
	}

	// Valid safe mapping.
	got, err := ParseAndValidateHostMappings(`[{"host":"/mnt/shared","container":"/workspace","read_only":true}]`)
	if err != nil {
		t.Fatalf("valid mapping failed: %v", err)
	}
	if len(got) != 1 || got[0].Host != "/mnt/shared" || got[0].Container != "/workspace" || !got[0].ReadOnly {
		t.Fatalf("unexpected parsed mapping: %+v", got)
	}

	// Forbidden / unsafe mappings.
	badCases := []struct {
		name string
		raw  string
	}{
		{"invalid json", `{not-json`},
		{"host root", `[{"host":"/","container":"/hostroot","read_only":false}]`},
		{"host root trailing slash", `[{"host":"///","container":"/hostroot","read_only":true}]`},
		{"docker sock", `[{"host":"/var/run/docker.sock","container":"/sock","read_only":false}]`},
		{"run docker sock", `[{"host":"/run/docker.sock","container":"/sock","read_only":false}]`},
		{"etc", `[{"host":"/etc","container":"/hostetc","read_only":true}]`},
		{"etc shadow", `[{"host":"/etc/shadow","container":"/shadow","read_only":true}]`},
		{"proc", `[{"host":"/proc","container":"/hostproc","read_only":true}]`},
		{"sys", `[{"host":"/sys","container":"/hostsys","read_only":true}]`},
		{"dev", `[{"host":"/dev","container":"/hostdev","read_only":true}]`},
		{"var", `[{"host":"/var","container":"/hostvar","read_only":true}]`},
		{"data", `[{"host":"/data","container":"/hostdata","read_only":true}]`},
		{"root home", `[{"host":"/root","container":"/hostroot","read_only":true}]`},
		{"usr", `[{"host":"/usr","container":"/hostusr","read_only":true}]`},
		{"traversal", `[{"host":"/mnt/../etc","container":"/workspace","read_only":true}]`},
		{"relative host", `[{"host":"relative/path","container":"/workspace","read_only":true}]`},
		{"relative container", `[{"host":"/mnt/data","container":"workspace","read_only":true}]`},
		{"container root", `[{"host":"/mnt/data","container":"/","read_only":true}]`},
		{"container entrypoint", `[{"host":"/mnt/data","container":"/entrypoint.sh","read_only":true}]`},
		{"container etc", `[{"host":"/mnt/data","container":"/etc/foo","read_only":true}]`},
	}

	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAndValidateHostMappings(tc.raw); err == nil {
				t.Fatalf("expected error for %s (%s), got nil", tc.name, tc.raw)
			}
		})
	}
}

func TestParseAndValidateEnvVars(t *testing.T) {
	for _, raw := range []string{"", "{}", "  {}  "} {
		got, err := ParseAndValidateEnvVars(raw)
		if err != nil {
			t.Fatalf("ParseAndValidateEnvVars(%q) unexpected error: %v", raw, err)
		}
		if len(got) != 0 {
			t.Fatalf("ParseAndValidateEnvVars(%q) = %v, want empty", raw, got)
		}
	}

	got, err := ParseAndValidateEnvVars(`{"FOO":"bar"}`)
	if err != nil || got["FOO"] != "bar" {
		t.Fatalf("valid env_vars failed: got=%v err=%v", got, err)
	}

	for _, bad := range []string{`not-json`, `{"A=B":"1"}`, `{"":"1"}`} {
		if _, err := ParseAndValidateEnvVars(bad); err == nil {
			t.Fatalf("expected error for %q, got nil", bad)
		}
	}
}
