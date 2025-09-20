package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type HandlerData struct {
	PackageName string
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run generate-handler.go <package-name> <output-file>")
		fmt.Println("Example: go run generate-handler.go compute /path/to/compute/handler.go")
		os.Exit(1)
	}

	packageName := os.Args[1]
	outputFile := os.Args[2]

	// Check if the handler file already exists
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("Handler file already exists, skipping: %s\n", outputFile)
		return
	}

	// Read the template file
	templatePath := filepath.Join("..", "templates", "handler.go.tmpl")
	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Printf("Error reading template file: %v\n", err)
		os.Exit(1)
	}

	// Parse the template
	tmpl, err := template.New("handler").Parse(string(tmplContent))
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

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

	data := HandlerData{
		PackageName: packageName,
	}

	if err := tmpl.Execute(file, data); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Generated handler file: %s\n", outputFile)
}
