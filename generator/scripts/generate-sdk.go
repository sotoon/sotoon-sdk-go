package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Module struct {
	ModuleName  string // e.g., "compute", "sotoon-kubernetes-engine"
	PackageName string // e.g., "compute", "sotoon_kubernetes_engine"
	ImportAlias string // e.g., "compute", "sotoon_kubernetes_engine"
	FieldName   string // e.g., "Compute", "Engine"
	VarName     string // e.g., "compute", "engine"
}

type SDKData struct {
	Modules []Module
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run generate-sdk.go <core-directory> <output-file>")
		fmt.Println("Example: go run generate-sdk.go ../../sdk/core ../../sdk/sdk.go")
		os.Exit(1)
	}

	coreDir := os.Args[1]
	outputFile := os.Args[2]

	// Discover modules in the core directory
	modules, err := discoverModules(coreDir)
	if err != nil {
		fmt.Printf("Error discovering modules: %v\n", err)
		os.Exit(1)
	}

	if len(modules) == 0 {
		fmt.Println("No modules found in core directory")
		os.Exit(1)
	}

	// Read the template file
	templatePath := filepath.Join("..", "templates", "sdk.go.tmpl")
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Printf("Error reading template file: %v\n", err)
		os.Exit(1)
	}

	// Parse the template
	tmpl, err := template.New("sdk").Parse(string(tmplContent))
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Execute template
	data := SDKData{
		Modules: modules,
	}

	if err := tmpl.Execute(file, data); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Generated SDK file: %s\n", outputFile)
	fmt.Printf("  Found %d modules: %s\n", len(modules), getModuleNames(modules))
}

func discoverModules(coreDir string) ([]Module, error) {
	var modules []Module

	entries, err := os.ReadDir(coreDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		moduleName := entry.Name()

		// Check if this directory has a handler.go file (indicating it's a valid module)
		handlerPath := filepath.Join(coreDir, moduleName, "handler.go")
		if _, err := os.Stat(handlerPath); os.IsNotExist(err) {
			continue
		}

		// Convert module name to package name (replace hyphens with underscores)
		packageName := strings.ReplaceAll(moduleName, "-", "_")

		// Create import alias (same as package name)
		importAlias := packageName

		// Create field name (capitalize and clean up)
		fieldName := createFieldName(moduleName)

		// Create variable name (lowercase version of field name)
		varName := strings.ToLower(string(fieldName[0])) + fieldName[1:]

		modules = append(modules, Module{
			ModuleName:  moduleName,
			PackageName: packageName,
			ImportAlias: importAlias,
			FieldName:   fieldName,
			VarName:     varName,
		})
	}

	return modules, nil
}

func createFieldName(moduleName string) string {
	// Handle special cases
	switch moduleName {
	case "sotoon-kubernetes-engine":
		return "Engine"
	case "compute":
		return "Compute"
	default:
		// Convert kebab-case to PascalCase
		parts := strings.Split(moduleName, "-")
		var result strings.Builder
		for _, part := range parts {
			if len(part) > 0 {
				result.WriteString(strings.ToUpper(string(part[0])) + part[1:])
			}
		}
		return result.String()
	}
}

func getModuleNames(modules []Module) string {
	var names []string
	for _, module := range modules {
		names = append(names, module.ModuleName)
	}
	return strings.Join(names, ", ")
}
