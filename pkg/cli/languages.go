package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
)

// LanguageInfo represents information about a supported language
type LanguageInfo struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	DisplayName      string   `json:"display_name"`
	SupportsGRPC     bool     `json:"supports_grpc"`
	FileExtensions   []string `json:"file_extensions"`
	Enabled          bool     `json:"enabled"`
	Stable           bool     `json:"stable"`
	Description      string   `json:"description"`
	DocumentationURL string   `json:"documentation_url"`
	PluginVersion    string   `json:"plugin_version"`
}

func newLanguagesCommand() *Command {
	cmd := &Command{
		Name:        "languages",
		Description: "Language management commands",
		Subcommands: make(map[string]*Command),
		Run:         runLanguages,
	}
	cmd.Subcommands["list"] = newLanguagesListCommand()
	cmd.Subcommands["show"] = newLanguagesShowCommand()
	return cmd
}

func runLanguages(args []string) error {
	// Handle subcommands
	if len(args) == 0 {
		return runLanguagesHelp(args)
	}

	langCmd := newLanguagesCommand()
	if subcmd, ok := langCmd.Subcommands[args[0]]; ok {
		return subcmd.Run(args[1:])
	}

	return fmt.Errorf("unknown languages subcommand: %s", args[0])
}

func runLanguagesHelp(args []string) error {
	fmt.Println("Usage: spoke languages <command> [args]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  list    List all supported languages")
	fmt.Println("  show    Show details for a specific language")
	fmt.Println("\nExamples:")
	fmt.Println("  spoke languages list")
	fmt.Println("  spoke languages show go")
	return nil
}

func newLanguagesListCommand() *Command {
	cmd := &Command{
		Name:        "list",
		Description: "List all supported languages",
		Flags:       flag.NewFlagSet("languages list", flag.ExitOnError),
		Run:         runLanguagesList,
	}

	cmd.Flags.String("registry", "http://localhost:8080", "Spoke registry URL")
	cmd.Flags.Bool("json", false, "Output in JSON format")

	return cmd
}

func newLanguagesShowCommand() *Command {
	cmd := &Command{
		Name:        "show",
		Description: "Show details for a specific language",
		Flags:       flag.NewFlagSet("languages show", flag.ExitOnError),
		Run:         runLanguagesShow,
	}

	cmd.Flags.String("registry", "http://localhost:8080", "Spoke registry URL")
	cmd.Flags.Bool("json", false, "Output in JSON format")

	return cmd
}

func runLanguagesList(args []string) error {
	cmd := newLanguagesListCommand()
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	registry := cmd.Flags.Lookup("registry").Value.String()
	outputJSON := cmd.Flags.Lookup("json").Value.String() == "true"

	// Make API request
	url := registry + "/api/v1/languages"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var languages []LanguageInfo
	if err := json.NewDecoder(resp.Body).Decode(&languages); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(languages)
	}

	// Pretty table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tVERSION\tGRPC\tSTABLE\tFILE EXTENSIONS")
	fmt.Fprintln(w, "──\t────\t───────\t────\t──────\t───────────────")

	for _, lang := range languages {
		grpcSupport := "✓"
		if !lang.SupportsGRPC {
			grpcSupport = "✗"
		}
		stable := "✓"
		if !lang.Stable {
			stable = "✗"
		}

		extensions := ""
		if len(lang.FileExtensions) > 0 {
			extensions = lang.FileExtensions[0]
			if len(lang.FileExtensions) > 1 {
				extensions += ", ..."
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			lang.ID,
			lang.Name,
			lang.PluginVersion,
			grpcSupport,
			stable,
			extensions,
		)
	}

	w.Flush()

	fmt.Printf("\nTotal: %d languages\n", len(languages))
	fmt.Println("\nUse 'spoke languages show <id>' for more details")

	return nil
}

func runLanguagesShow(args []string) error {
	cmd := newLanguagesShowCommand()
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	registry := cmd.Flags.Lookup("registry").Value.String()
	outputJSON := cmd.Flags.Lookup("json").Value.String() == "true"

	// Get remaining args (language ID)
	remainingArgs := cmd.Flags.Args()
	if len(remainingArgs) == 0 {
		return fmt.Errorf("language ID required. Usage: spoke languages show <id>")
	}
	languageID := remainingArgs[0]

	// Make API request
	url := fmt.Sprintf("%s/api/v1/languages/%s", registry, languageID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var lang LanguageInfo
	if err := json.NewDecoder(resp.Body).Decode(&lang); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(lang)
	}

	// Pretty output
	fmt.Printf("Language: %s\n", lang.DisplayName)
	fmt.Printf("ID: %s\n", lang.ID)
	fmt.Printf("Version: %s\n", lang.PluginVersion)
	fmt.Printf("gRPC Support: %v\n", lang.SupportsGRPC)
	fmt.Printf("Stable: %v\n", lang.Stable)
	fmt.Printf("Enabled: %v\n", lang.Enabled)
	fmt.Printf("\nDescription:\n  %s\n", lang.Description)
	fmt.Printf("\nFile Extensions:\n")
	for _, ext := range lang.FileExtensions {
		fmt.Printf("  - %s\n", ext)
	}
	fmt.Printf("\nDocumentation: %s\n", lang.DocumentationURL)

	return nil
}
