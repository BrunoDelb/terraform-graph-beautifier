package main

import (
	"flag"
	"fmt"
	"github.com/pcasteran/terraform-graph-beautifier/assets"
	"github.com/pcasteran/terraform-graph-beautifier/cytoscape"
	"github.com/pcasteran/terraform-graph-beautifier/graphviz"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	outputTypeCytoscapeJSON = "cyto-json"
	outputTypeCytoscapeHTML = "cyto-html"
	outputTypeGraphviz      = "graphviz"
)

func getWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Base(dir)
}

func main() {
	// Prepare command line options.
	inputFilePath := flag.String("input", "", "Path of the input Graphviz file to read, if not set 'stdin' is used")
	outputType := flag.String("output-type", outputTypeCytoscapeHTML, fmt.Sprintf("Type of output, can be one the following : %s, %s, %s", outputTypeCytoscapeJSON, outputTypeCytoscapeHTML, outputTypeGraphviz))
	outputFilePath := flag.String("output", "", "Path of the output file to write, if not set 'stdout' is used")
	debug := flag.Bool("debug", false, "Print debugging information to stderr")
	printVersion := flag.Bool("v", false, "Print command version and exit")
	// Input reading options.
	var excludePatterns arrayFlags
	flag.Var(&excludePatterns, "exclude", "Pattern (regexp) of the resource to filter out (can be repeated multiple times)")
	keepTfJunk := flag.Bool("keep-tf-junk", false, "Do not remove the \"junk\" nodes and edges generated by 'terraform graph' (default false)")
	// Output writing options.
	graphName := flag.String("graph-name", getWorkingDir(), "Name of the output graph, defaults to working directory name")
	embedModules := flag.Bool("embed-modules", true, "Embed a module sub-graph inside its parent if true; otherwise the two modules are siblings and an edge is drawn from the parent to the child")
	cytoHTMLTemplatePath := flag.String("cyto-html-template", "", fmt.Sprintf("Path of the HTML template to use for Cytoscape.js rendering (output-type=\"%s\"), if not set a default one is used", outputTypeCytoscapeHTML))

	// Parse command line arguments.
	flag.Parse()

	if *printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// Configure logging.
	// Default level for this example is info, unless debug flag is present.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load the graph from the specified input.
	inputFile := os.Stdin
	var err error
	if *inputFilePath != "" {
		// Read from the specified file.
		inputFile, err = os.Open(*inputFilePath)
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot open the file for reading")
		}
		defer func() {
			if err := inputFile.Close(); err != nil {
				log.Fatal().Err(err).Msg("Cannot close the file after reading")
			}
		}()
	}
	graph := graphviz.LoadGraph(inputFile, *keepTfJunk, excludePatterns)

	// Write the result to the specified output.
	outputFile := os.Stdout
	if *outputFilePath != "" {
		// Write to the specified file.
		outputFile, err = os.Create(*outputFilePath)
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot open the file for writing")
		}
		defer func() {
			if err := outputFile.Close(); err != nil {
				log.Fatal().Err(err).Msg("Cannot close the file after writing")
			}
		}()
	}

	switch *outputType {
	case outputTypeCytoscapeJSON:
		log.Debug().Msg("Output graph data to Cytoscape.js JSON format")
		cytoscape.WriteGraphJSON(outputFile, graph, &cytoscape.RenderingOptions{
			GraphName:    *graphName,
			EmbedModules: *embedModules,
		})
	case outputTypeCytoscapeHTML:
		log.Debug().Msg("Output graph to HTML")

		// Open HTML template file.
		var template http.File = nil
		if *cytoHTMLTemplatePath != "" {
			// Use the specified template file.
			template, err = os.Open(*cytoHTMLTemplatePath)
		} else {
			// Use the default template file.
			template, err = assets.Templates.Open("index.gohtml")
		}
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot open the HTML template file for reading")
		}
		defer func() {
			if err := template.Close(); err != nil {
				log.Fatal().Err(err).Msg("Cannot close the file after reading")
			}
		}()

		cytoscape.WriteGraphHTML(outputFile, graph, &cytoscape.RenderingOptions{
			GraphName:    *graphName,
			EmbedModules: *embedModules,
			HTMLTemplate: template,
		})
	case outputTypeGraphviz:
		log.Debug().Msg("Output graph data to Graphviz Dot format")
		graphviz.WriteGraph(outputFile, graph, &graphviz.RenderingOptions{
			GraphName:    *graphName,
			EmbedModules: *embedModules,
		})
	default:
		log.Fatal().Err(err).Msg(fmt.Sprintf("Invalid output type : %s", *outputType))
	}
}
