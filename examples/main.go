package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/panyam/templar"
)

type User struct {
	ID   int
	Name string
}

type Update struct {
	Title string
	Date  string
}

type FeaturedContent struct {
	Title       string
	Description string
	URL         string
}

func main() {
	// Create a template group
	group := templar.NewTemplateGroup()

	// Create a filesystem loader that searches multiple directories
	group.Loader = templar.NewFileSystemLoader(
		"templates/",
		"templates/shared/",
	)

	// Add custom functions
	group.AddFuncs(map[string]any{
		"currentYear": func() int {
			return time.Now().Year()
		},
		// Helper function to create dictionaries in templates
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]any)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		// Default value helper
		"default": func(def, val any) any {
			if val == nil || val == "" {
				return def
			}
			return val
		},
	})

	// Example of using a loader list with fallbacks
	err := os.MkdirAll("./output", 0755)
	if err != nil {
		log.Fatal("Could not create directory: ", err)
		panic(err)
	}

	basicExample(group, openFile("./output/basic.html"))

	exampleLoaderList(openFile("./output/list.html"))

	// Example of conditional template loading based on device
	exampleConditionalLoading(true, group, openFile("./output/conditional_mobile.html"))

	exampleConditionalLoading(false, group, openFile("./output/conditional_desktop.html"))

	// Example of dynamic template creation
	exampleDynamicTemplate(group, openFile("./output/dynamic.html"))

	// Example of namespace and extend features
	exampleNamespaceAndExtend(group, openFile("./output/namespace_demo.html"))
}

func openFile(outfile string) io.Writer {
	out, err := os.Create(outfile)
	if err != nil {
		panic(err)
	}
	return out
}

// Basic example showing template loading and rendering
func basicExample(group *templar.TemplateGroup, w io.Writer) {
	// Load a root template (dependencies handled automatically)
	rootTemplate := group.MustLoad("pages/homepage.tmpl", "")

	// Prepare data for the template
	data := map[string]any{
		"Title": "Home Page",
		"User": User{
			ID:   1,
			Name: "John Doe",
		},
		"Updates": []Update{
			{Title: "New Feature Released", Date: "2023-06-15"},
			{Title: "System Maintenance", Date: "2023-06-10"},
			{Title: "Welcome to our New Site", Date: "2023-06-01"},
		},
		"Featured": FeaturedContent{
			Title:       "Summer Sale",
			Description: "Get 20% off on all products until July 31st!",
			URL:         "/summer-sale",
		},
	}

	// Render the template to stdout (for this example)
	fmt.Println("Rendering template...")
	if err := group.RenderHtmlTemplate(w, rootTemplate[0], "", data, nil); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

// Example showing how to use a loader list
func exampleLoaderList(w io.Writer) {
	// Create a list of loaders to search in order
	loaderList := &templar.LoaderList{}

	// Add loaders in priority order
	loaderList.AddLoader(templar.NewFileSystemLoader("app/templates/"))
	loaderList.AddLoader(templar.NewFileSystemLoader("shared/templates/"))

	// Set a default loader as final fallback
	loaderList.DefaultLoader = templar.NewFileSystemLoader("default/templates/")

	fmt.Println("Loader list configured with fallback options")
}

// Example showing conditional template loading
func exampleConditionalLoading(isMobile bool, group *templar.TemplateGroup, w io.Writer) {
	// Choose template based on device
	templatePath := "desktop/homepage.tmpl"
	if isMobile {
		templatePath = "mobile/homepage.tmpl"
	}

	if isMobile {
		fmt.Printf("Loading template for mobile device...\n")
	} else {
		fmt.Printf("Loading template for desktop device...\n")
	}
	deviceType := "desktop"
	if isMobile {
		deviceType = "mobile"
	}
	fmt.Printf("Loading template for %s device...\n", deviceType)

	// Load the appropriate template
	tmpl, err := group.Loader.Load(templatePath, "")
	if err != nil {
		fmt.Printf("Error loading template: %v\n", err)
		return
	}

	// Use the loaded template
	if len(tmpl) > 0 {
		fmt.Printf("Successfully loaded %s\n", templatePath)

		// Example data
		data := map[string]any{
			"Title": "Home Page",
			"User": User{
				ID:   1,
				Name: "John Doe",
			},
		}

		// In a real application, you would render to a response writer
		// For this example, we'll just note that we could render it
		fmt.Println("Template ready for rendering")

		group.RenderHtmlTemplate(w, tmpl[0], "", data, nil)
	}
}

// Example showing dynamic template creation
func exampleDynamicTemplate(group *templar.TemplateGroup, w io.Writer) {
	// Create a template on the fly
	dynamicTemplate := &templar.Template{
		Name:      "dynamic-template",
		RawSource: []byte(`Hello, {{.Name}}!`),
	}

	// Prepare data
	data := map[string]any{
		"Name": "World",
	}

	// Render the dynamic template
	fmt.Println("Rendering dynamic template:")
	err := group.RenderTextTemplate(w, dynamicTemplate, "", data, nil)
	if err != nil {
		fmt.Printf("Error rendering dynamic template: %v\n", err)
	}
}

// Example showing namespace and extend features
func exampleNamespaceAndExtend(group *templar.TemplateGroup, w io.Writer) {
	fmt.Println("Loading namespace demo template...")

	// Load the namespace demo template which demonstrates:
	// 1. {{# namespace "Theme" "themes/bootstrap.tmpl" #}} - Import theme into "Theme" namespace
	// 2. {{# namespace "UI" "widgets/buttons.tmpl" #}} - Import widgets into "UI" namespace
	// 3. {{# extend "Theme:layout" "MyLayout" ... #}} - Extend base layout with overrides
	// 4. Using namespaced templates like {{ template "UI:button" ... }}
	rootTemplate := group.MustLoad("pages/namespace_demo.tmpl", "")

	// Prepare data for the template
	data := map[string]any{
		"Title": "Namespace & Extend Demo",
	}

	// Render the template
	fmt.Println("Rendering namespace demo...")
	if err := group.RenderHtmlTemplate(w, rootTemplate[0], "", data, nil); err != nil {
		fmt.Printf("Error rendering namespace demo: %v\n", err)
	}
	fmt.Println("Namespace demo rendered successfully!")
}
