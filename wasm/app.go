package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/Skarlso/crd-to-sample-yaml/pkg"
)

// crdView is the main component to display a rendered CRD.
type crdView struct {
	app.Compo

	content []byte
	comment bool
}

// Version wraps a top level version resource which contains the underlying openAPIV3Schema.
type Version struct {
	Version     string
	Kind        string
	Group       string
	Properties  []*Property
	Description string
	YAML        string
}

// Property builds up a Tree structure of embedded things.
type Property struct {
	Name        string
	Description string
	Type        string
	Nullable    bool
	Patterns    string
	Format      string
	Indent      int
	Version     string
	Default     string
	Required    bool
	Properties  []*Property
}

func (h *crdView) buildError(err error) app.UI {
	return app.Div().Class("alert alert-danger").Role("alert").Body(
		app.Span().Class("closebtn").Body(app.Text("×")),
		app.H4().Class("alert-heading").Text("Oops!"),
		app.Text(err.Error()))
}

// The Render method is where the component appearance is defined. Here, a
// "Hello World!" is displayed as a heading.
func (h *crdView) Render() app.UI {
	crd := &v1beta1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(h.content, crd); err != nil {
		return h.buildError(err)
	}

	versions := make([]Version, 0)
	for _, version := range crd.Spec.Versions {
		out, err := parseCRD(version.Schema.OpenAPIV3Schema.Properties, version.Name, version.Schema.OpenAPIV3Schema.Required)
		if err != nil {
			return h.buildError(err)
		}
		var buffer []byte
		buf := bytes.NewBuffer(buffer)
		if err := pkg.ParseProperties(crd.Spec.Group, version.Name, crd.Spec.Names.Kind, version.Schema.OpenAPIV3Schema.Properties, buf, 0, false, h.comment); err != nil {
			return h.buildError(err)
		}
		versions = append(versions, Version{
			Version:     version.Name,
			Properties:  out,
			Kind:        crd.Spec.Names.Kind,
			Group:       crd.Spec.Group,
			Description: version.Schema.OpenAPIV3Schema.Description,
			YAML:        buf.String(),
		})
	}

	wrapper := app.Div().Class("content-wrapper")
	container := app.Div().Class("container")
	container.Body(app.Range(versions).Slice(func(i int) app.UI {
		div := app.Div().Class("versions")
		version := versions[i]
		yamlContent := app.Div().Class("accordion").ID("yaml-accordion-" + version.Version).Body(
			app.Div().Class("accordion-item").Body(
				app.H2().Class("accordion-header").Body(
					app.Button().Class("accordion-button").Type("button").DataSets(
						map[string]any{
							"bs-toggle": "collapse",
							"bs-target": "#yaml-accordion-collapse-" + version.Version}).
						Aria("expanded", "false").
						Aria("controls", "yaml-accordion-collapse-"+version.Version).
						Body(app.Text("Details")),
				),
				app.Div().Class("accordion-collapse collapse").ID("yaml-accordion-collapse-"+version.Version).DataSet("bs-parent", "#yaml-accordion-"+version.Version).Body(
					app.Div().Class("accordion-body").Body(
						app.Div().Class("container").Body(
							app.Div().Class("row").Body(
								app.Div().Class("col-9").Body(
									app.Pre().Body(
										app.Code().Class("language-yaml").ID("yaml-sample-"+version.Version).Body(
											app.Text(version.YAML),
										)),
								),
								app.Div().Class("col-3").Body(
									app.Button().Class("clippy-"+strconv.Itoa(i)).DataSet("clipboard-target", "#yaml-sample-"+version.Version).Body(
										app.Script().Text(fmt.Sprintf("new ClipboardJS('.clippy-%d');", i)),
										app.I().Class("fa fa-clipboard"),
									),
								),
							),
						),
					),
				),
			),
		)
		div.Body(
			app.H1().Body(
				app.P().Body(app.Text(fmt.Sprintf(
					`Version: %s/%s`,
					version.Group,
					version.Version,
				))),
				app.P().Body(app.Text(fmt.Sprintf("Kind: %s", version.Kind)))),
			app.P().Body(app.Text(version.Description)),
			app.P().Body(app.Text("Generated YAML sample:")),
			yamlContent,
			app.H1().Text(version.Version),
			app.Div().Class("accordion").ID("version-accordion-"+version.Version).Body(
				render(app.Div().Class("accordion-item"), version.Properties, "version-accordion-"+version.Version, 0),
			),
		)
		return div
	}))

	return wrapper.Body(
		app.Script().Text("hljs.highlightAll();"),
		container,
	)
}

var borderOpacity = map[int]string{
	0: "border border-secondary-subtle",
	1: "border border-secondary-subtle border-opacity-75",
	2: "border border-secondary-subtle border-opacity-50",
	3: "border border-secondary-subtle border-opacity-25",
	4: "border border-secondary-subtle border-opacity-10",
}

func render(d app.UI, p []*Property, accordionID string, depth int) app.UI {
	borderOpacity, ok := borderOpacity[depth]
	if !ok {
		borderOpacity = ""
	}

	var elements []app.UI
	for _, prop := range p {
		// add the parent first
		headerElements := []app.UI{
			app.Div().Class("col").Body(app.Text(prop.Name)),
			app.Div().Class("text-muted col").Text(prop.Type),
		}

		if prop.Required {
			headerElements = append(headerElements, app.Div().Class("text-bg-primary").Class("col").Text("required"))
		}
		if prop.Format != "" {
			headerElements = append(headerElements, app.Div().Class("col").Text(prop.Format))
		}
		if prop.Default != "" {
			headerElements = append(headerElements, app.Div().Class("col").Text(prop.Default))
		}
		if prop.Patterns != "" {
			headerElements = append(headerElements, app.Div().Class("col").Class("fst-italic").Text(prop.Patterns))
		}

		headerContainer := app.Div().Class("container").Body(
			// Both rows are important here to produce the desired outcome.
			app.Div().Class("row").Body(
				app.P().Class("fw-bold").Class("row").Body(
					headerElements...,
				),
				app.Div().Class("row").Class("text-break").Body(app.Text(prop.Description)),
			),
		)

		button := app.Button().ID("accordion-button-id-"+prop.Name+accordionID).Class("accordion-button").Type("button").DataSets(
			map[string]any{
				"bs-toggle": "collapse",
				"bs-target": "#accordion-collapse-for-" + prop.Name + accordionID}).
			Aria("expanded", "false").
			Aria("controls", "accordion-collapse-for-"+prop.Name+accordionID).
			Body(
				headerContainer,
			)

		if len(prop.Properties) != 0 {
			button.Class("bg-success-subtle")
		}

		header := app.H2().Class("accordion-header").Class(borderOpacity).Body(button)

		elements = append(elements, header)

		// The next section can be skipped if there are no child properties.
		if len(prop.Properties) == 0 {
			continue
		}

		accordionDiv := app.Div().Class("accordion-collapse collapse").ID("accordion-collapse-for-"+prop.Name+accordionID).DataSet("bs-parent", "#"+accordionID)
		accordionBody := app.Div().Class("accordion-body")

		var bodyElements []app.UI

		// add any children that the parent has
		if len(prop.Properties) > 0 {
			element := render(app.Div().ID(prop.Name).Class("accordion-item"), prop.Properties, "accordion-collapse-for-"+prop.Name+accordionID, depth+1)
			bodyElements = append(bodyElements, element)
		}

		accordionBody.Body(bodyElements...)
		accordionDiv.Body(accordionBody)
		elements = append(elements, accordionDiv)
	}

	// add all the elements and return the div
	switch t := d.(type) {
	case app.HTMLDiv:
		t.Body(elements...)
		d = t
	}

	return d
}

// parseCRD takes the properties and constructs a linked list out of the embedded properties that the recursive
// template can call and construct linked divs.
func parseCRD(properties map[string]v1beta1.JSONSchemaProps, version string, requiredList []string) ([]*Property, error) {
	var (
		sortedKeys []string
		output     []*Property
	)
	for k := range properties {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		// Create the Property with the values necessary.
		// Check if there are properties for it in Properties or in Array -> Properties.
		// If yes, call parseCRD and add the result to the created properties Properties list.
		// If not, or if we are done, add this new property to the list of properties and return it.
		v := properties[k]
		required := false
		for _, item := range requiredList {
			if item == k {
				required = true
				break
			}
		}
		p := &Property{
			Name:        k,
			Type:        v.Type,
			Description: v.Description,
			Patterns:    v.Pattern,
			Format:      v.Format,
			Nullable:    v.Nullable,
			Version:     version,
			Required:    required,
		}
		if v.Default != nil {
			p.Default = string(v.Default.Raw)
		}

		if len(properties[k].Properties) > 0 && properties[k].AdditionalProperties == nil {
			requiredList = v.Required
			out, err := parseCRD(properties[k].Properties, version, requiredList)
			if err != nil {
				return nil, err
			}
			p.Properties = out
		} else if properties[k].Type == "array" && properties[k].Items.Schema != nil && len(properties[k].Items.Schema.Properties) > 0 {
			requiredList = v.Required
			out, err := parseCRD(properties[k].Items.Schema.Properties, version, requiredList)
			if err != nil {
				return nil, err
			}
			p.Properties = out
		} else if properties[k].AdditionalProperties != nil {
			requiredList = v.Required
			out, err := parseCRD(properties[k].AdditionalProperties.Schema.Properties, version, requiredList)
			if err != nil {
				return nil, err
			}
			p.Properties = out
		}
		output = append(output, p)
	}
	return output, nil
}
