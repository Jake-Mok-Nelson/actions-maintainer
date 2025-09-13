package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/workflow"
	"github.com/Jake-Mok-Nelson/actions-maintainer/internal/actions"
)

func main() {
	// Read test workflow file
	content, err := ioutil.ReadFile("/tmp/test-workflow.yml")
	if err != nil {
		log.Fatal(err)
	}

	// Parse workflow
	actionRefs, err := workflow.ParseWorkflow(string(content), ".github/workflows/ci.yml", "test/repo")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d action references:\n", len(actionRefs))
	for _, ref := range actionRefs {
		fmt.Printf("- %s@%s (reusable: %v) in %s\n", ref.Repository, ref.Version, ref.IsReusable, ref.Context)
	}

	// Test action analysis
	manager := actions.NewManager()
	issues := manager.AnalyzeActions(actionRefs)
	
	fmt.Printf("\nFound %d issues:\n", len(issues))
	for _, issue := range issues {
		fmt.Printf("- %s: %s (%s severity)\n", issue.Repository, issue.Description, issue.Severity)
	}
}