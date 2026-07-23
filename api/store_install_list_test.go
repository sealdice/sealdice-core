package api

import (
	"testing"

	"sealdice-core/dice"
)

func TestInstallStorePackageBatchRetriesDependencies(t *testing.T) {
	targets := []*dice.StorePackage{
		{ID: "alice/app", Version: "1.0.0"},
		{ID: "alice/base", Version: "1.0.0"},
	}
	results := []storeInstallListItemResult{
		{ID: targets[0].ID, Version: targets[0].Version},
		{ID: targets[1].ID, Version: targets[1].Version},
	}
	pending := []pendingStoreInstall{
		{target: targets[0], resultIndex: 0},
		{target: targets[1], resultIndex: 1},
	}
	installed := map[string]bool{}
	attempts := map[string]int{}

	installStorePackageBatch(results, pending, func(target *dice.StorePackage, _ bool) (string, error) {
		attempts[target.ID]++
		if target.ID == "alice/app" && !installed["alice/base"] {
			return "", &dice.DependencyError{
				PackageID:   target.ID,
				MissingDeps: []string{"alice/base"},
			}
		}
		installed[target.ID] = true
		return "installed", nil
	})

	if results[0].Status != "installed" || results[1].Status != "installed" {
		t.Fatalf("results = %#v", results)
	}
	if attempts["alice/app"] != 2 || attempts["alice/base"] != 1 {
		t.Fatalf("attempts = %#v", attempts)
	}
}

func TestInstallStorePackageBatchReportsUnresolvedDependency(t *testing.T) {
	target := &dice.StorePackage{ID: "alice/app", Version: "1.0.0"}
	results := []storeInstallListItemResult{{ID: target.ID, Version: target.Version}}
	pending := []pendingStoreInstall{{target: target, resultIndex: 0}}

	installStorePackageBatch(results, pending, func(target *dice.StorePackage, _ bool) (string, error) {
		return "", &dice.DependencyError{
			PackageID:   target.ID,
			MissingDeps: []string{"alice/missing"},
		}
	})

	if results[0].Status != "failed" || results[0].Message == "" {
		t.Fatalf("result = %#v", results[0])
	}
}
