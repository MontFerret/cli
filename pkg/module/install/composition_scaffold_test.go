package install

import "testing"

func TestScaffoldPackageNameNormalizesModuleLeaf(t *testing.T) {
	for _, test := range []struct {
		modulePath string
		want       string
	}{
		{modulePath: "montferret.com/xproject", want: "xproject"},
		{modulePath: "example.com/data-tools", want: "data_tools"},
		{modulePath: "example.com/123", want: "project_123"},
		{modulePath: "example.com/type", want: "typeproject"},
	} {
		t.Run(test.modulePath, func(t *testing.T) {
			if got := scaffoldPackageName(test.modulePath); got != test.want {
				t.Fatalf("scaffoldPackageName(%q) = %q, want %q", test.modulePath, got, test.want)
			}
		})
	}
}
