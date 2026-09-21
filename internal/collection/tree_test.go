package collection_test

import (
	"os"
	"path/filepath"
	"testing"

	"apitool/internal/collection"
	"apitool/internal/model"
)

func TestBuildTreeUsesRequestPathAsStableIDAndKeepsInvalidSibling(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "payments/list.yaml", "name: List payments\nmethod: GET\nrequest:\n  url: https://api.example.test/payments\n")
	writeDefinition(t, root, "payments/broken.yaml", "name: Broken\nmethod: GET\nrequest: [not-a-request]\n")

	tree, diagnostics := collection.BuildTree(root)
	if _, ok := tree.Requests["payments/list"]; !ok {
		t.Fatalf("BuildTree() requests = %#v, want payments/list", tree.Requests)
	}
	if _, ok := tree.Invalid["payments/broken"]; !ok {
		t.Fatalf("BuildTree() invalid = %#v, want payments/broken", tree.Invalid)
	}
	if len(diagnostics) == 0 {
		t.Fatal("BuildTree() diagnostics = none, want invalid request diagnostic")
	}
}

func TestBuildTreeMarksParsedRequestInvalidWhenValidationFails(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "payments/missing-url.yaml", "name: Missing URL\nmethod: GET\nrequest: {}\n")

	tree, diagnostics := collection.BuildTree(root)
	if _, ok := tree.Requests["payments/missing-url"]; ok {
		t.Fatalf("BuildTree() requests = %#v, did not exclude invalid request", tree.Requests)
	}
	if _, ok := tree.Invalid["payments/missing-url"]; !ok {
		t.Fatalf("BuildTree() invalid = %#v, want missing-url request", tree.Invalid)
	}
	if len(diagnostics) == 0 {
		t.Fatal("BuildTree() diagnostics = none, want missing URL validation diagnostic")
	}
}

func TestBuildTreePreservesNestedGroupChainAndGroupAuth(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "admin/_group.yaml", "name: Admin\nauth: none\n")
	writeDefinition(t, root, "admin/refunds/_group.yaml", "name: Refunds\nauth:\n  type: oauth2\n  grant: client_credentials\n  scopes: refunds.read refunds.write\n")
	writeDefinition(t, root, "admin/refunds/list.yaml", "name: List refunds\nmethod: GET\nrequest:\n  url: https://api.example.test/refunds\n")

	tree, diagnostics := collection.BuildTree(root)
	if len(diagnostics) != 0 {
		t.Fatalf("BuildTree() diagnostics = %#v, want none", diagnostics)
	}
	request, ok := tree.Requests["admin/refunds/list"]
	if !ok {
		t.Fatalf("BuildTree() requests = %#v, want admin/refunds/list", tree.Requests)
	}
	if got, want := groupIDs(request.Groups), []string{"admin", "admin/refunds"}; !equalGroupIDs(got, want) {
		t.Fatalf("request group chain = %#v, want %#v", got, want)
	}
	if !request.Groups[0].Group.Auth.None {
		t.Fatalf("outer group auth = %#v, want auth none", request.Groups[0].Group.Auth)
	}
	if got, want := request.Groups[1].Group.Auth.Scopes, []string{"refunds.read", "refunds.write"}; !equalGroupIDs(got, want) {
		t.Fatalf("nested group auth scopes = %#v, want %#v", got, want)
	}
}

func TestBuildTreeAppliesGroupMetadataWhenRequestSortsBeforeGroupFile(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "admin/A-list.yaml", "name: List admin\nmethod: GET\nrequest:\n  url: https://api.example.test/admin\n")
	writeDefinition(t, root, "admin/_group.yaml", "name: Admin\nauth: none\n")

	tree, diagnostics := collection.BuildTree(root)
	if len(diagnostics) != 0 {
		t.Fatalf("BuildTree() diagnostics = %#v, want none", diagnostics)
	}
	request := tree.Requests["admin/A-list"]
	if len(request.Groups) != 1 || request.Groups[0].Group.Auth == nil || !request.Groups[0].Group.Auth.None {
		t.Fatalf("request group chain = %#v, want group auth none", request.Groups)
	}
}

func TestBuildTreePreservesYAMLSuffixInGroupDirectoryID(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "admin.yaml/_group.yaml", "name: Admin\nauth: none\n")
	writeDefinition(t, root, "admin.yaml/list.yaml", "name: List admin\nmethod: GET\nrequest:\n  url: https://api.example.test/admin\n")

	tree, diagnostics := collection.BuildTree(root)
	if len(diagnostics) != 0 {
		t.Fatalf("BuildTree() diagnostics = %#v, want none", diagnostics)
	}
	request, ok := tree.Requests["admin.yaml/list"]
	if !ok {
		t.Fatalf("BuildTree() requests = %#v, want admin.yaml/list", tree.Requests)
	}
	if got, want := groupIDs(request.Groups), []string{"admin.yaml"}; !equalGroupIDs(got, want) {
		t.Fatalf("request group chain = %#v, want %#v", got, want)
	}
	if request.Groups[0].Group.Auth == nil || !request.Groups[0].Group.Auth.None {
		t.Fatalf("request group auth = %#v, want auth none", request.Groups[0].Group.Auth)
	}
}

func TestBuildTreeMarksInvalidYAMLSuffixGroupAuthAndDependentRequest(t *testing.T) {
	for _, test := range []struct {
		name       string
		auth       string
		diagnostic string
	}{
		{name: "unsupported type", auth: "type: basic", diagnostic: "auth_type_unsupported"},
		{name: "unsupported grant", auth: "type: oauth2\n  grant: authorization_code", diagnostic: "auth_grant_unsupported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := collectionRoot(t)
			writeDefinition(t, root, "admin.yaml/_group.yaml", "auth:\n  "+test.auth+"\n")
			writeDefinition(t, root, "admin.yaml/list.yaml", "name: List admin\nmethod: GET\nrequest:\n  url: https://api.example.test/admin\n")

			tree, diagnostics := collection.BuildTree(root)
			if _, ok := tree.Invalid["admin.yaml"]; !ok {
				t.Fatalf("BuildTree() invalid = %#v, want invalid admin.yaml group", tree.Invalid)
			}
			if _, ok := tree.Invalid["admin.yaml/list"]; !ok {
				t.Fatalf("BuildTree() invalid = %#v, want invalid dependent request", tree.Invalid)
			}
			request, ok := tree.Requests["admin.yaml/list"]
			if !ok {
				t.Fatalf("BuildTree() requests = %#v, want retained dependent request", tree.Requests)
			}
			if !hasDiagnosticCode(request.Diagnostics, test.diagnostic) {
				t.Fatalf("request diagnostics = %#v, want %q", request.Diagnostics, test.diagnostic)
			}
			if !hasDiagnosticCode(diagnostics, test.diagnostic) {
				t.Fatalf("tree diagnostics = %#v, want %q", diagnostics, test.diagnostic)
			}
		})
	}
}

func TestBuildTreeSortsGroupAndRequestIDs(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "zebra/list.yaml", "name: Zebra\nmethod: GET\nrequest:\n  url: https://api.example.test/zebra\n")
	writeDefinition(t, root, "admin/list.yaml", "name: Admin\nmethod: GET\nrequest:\n  url: https://api.example.test/admin\n")

	tree, diagnostics := collection.BuildTree(root)
	if len(diagnostics) != 0 {
		t.Fatalf("BuildTree() diagnostics = %#v, want none", diagnostics)
	}
	if got, want := tree.RequestIDs, []string{"admin/list", "zebra/list"}; !equalGroupIDs(got, want) {
		t.Fatalf("request IDs = %#v, want %#v", got, want)
	}
	if got, want := groupIDs(tree.Groups), []string{"admin", "zebra"}; !equalGroupIDs(got, want) {
		t.Fatalf("groups = %#v, want %#v", got, want)
	}
}

func TestBuildTreeRetainsInvalidGroupAndMarksDependentRequest(t *testing.T) {
	root := collectionRoot(t)
	writeDefinition(t, root, "admin/_group.yaml", "auth: invalid\n")
	writeDefinition(t, root, "admin/list.yaml", "name: List admin\nmethod: GET\nrequest:\n  url: https://api.example.test/admin\n")

	tree, diagnostics := collection.BuildTree(root)
	if _, ok := tree.Invalid["admin"]; !ok {
		t.Fatalf("BuildTree() invalid = %#v, want admin group", tree.Invalid)
	}
	if _, ok := tree.Requests["admin/list"]; !ok {
		t.Fatalf("BuildTree() requests = %#v, want valid sibling request", tree.Requests)
	}
	request := tree.Requests["admin/list"]
	if len(request.Diagnostics) == 0 {
		t.Fatal("dependent request diagnostics = none, want invalid group diagnostic")
	}
	if _, ok := tree.Invalid["admin/list"]; !ok {
		t.Fatalf("BuildTree() invalid = %#v, want dependent request marker", tree.Invalid)
	}
	if len(diagnostics) == 0 {
		t.Fatal("BuildTree() diagnostics = none, want invalid group diagnostic")
	}
}

func collectionRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "payments")
}

func writeDefinition(t *testing.T, collectionRoot, relativePath, content string) {
	t.Helper()
	path := filepath.Join(collectionRoot, ".api", "requests", relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func groupIDs(groups []collection.GroupNode) []string {
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	return ids
}

func equalGroupIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func hasDiagnosticCode(diagnostics []model.Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == want {
			return true
		}
	}
	return false
}
