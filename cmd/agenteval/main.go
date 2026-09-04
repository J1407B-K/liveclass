package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	agenteval "liveclass/internal/rpc/agent/eval"
)

type comparison struct {
	GeneratedAt string             `json:"generated_at"`
	CasesFile   string             `json:"cases_file"`
	Reports     []agenteval.Report `json:"reports"`
	Note        string             `json:"note"`
}

func main() {
	casesPath := flag.String("cases", "eval/cases.jsonl", "gold cases JSONL")
	predictionArgs := flag.String("predictions", "", "comma-separated variant=predictions.jsonl")
	category := flag.String("category", "", "only evaluate cases whose metadata.category matches")
	output := flag.String("output", "", "output report JSON (stdout when empty)")
	flag.Parse()
	if *predictionArgs == "" {
		fatal("predictions is required; example: v1=eval/runs/v1.jsonl,v2=eval/runs/v2.jsonl")
	}
	casesFile, err := os.Open(*casesPath)
	if err != nil {
		fatal(err.Error())
	}
	defer casesFile.Close()
	cases, err := agenteval.ReadJSONL[agenteval.Case](casesFile)
	if err != nil {
		fatal(err.Error())
	}
	if err := agenteval.ValidateCases(cases); err != nil {
		fatal(err.Error())
	}
	if *category != "" {
		filtered := cases[:0]
		for _, c := range cases {
			if value, _ := c.Metadata["category"].(string); value == *category {
				filtered = append(filtered, c)
			}
		}
		cases = filtered
		if len(cases) == 0 {
			fatal("no cases matched category " + *category)
		}
	}
	result := comparison{GeneratedAt: time.Now().UTC().Format(time.RFC3339), CasesFile: *casesPath, Note: "All metrics are computed from supplied prediction files; no model outputs are synthesized."}
	for _, item := range strings.Split(*predictionArgs, ",") {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			fatal("invalid predictions entry: " + item)
		}
		file, err := os.Open(parts[1])
		if err != nil {
			fatal(err.Error())
		}
		predictions, readErr := agenteval.ReadJSONL[agenteval.Prediction](file)
		file.Close()
		if readErr != nil {
			fatal(readErr.Error())
		}
		if err := agenteval.ValidatePredictions(predictions); err != nil {
			fatal(parts[0] + ": " + err.Error())
		}
		report := agenteval.Evaluate(parts[0], cases, predictions)
		if len(report.MissingPredictions) > 0 {
			fatal(fmt.Sprintf("%s: missing %d predictions", parts[0], len(report.MissingPredictions)))
		}
		result.Reports = append(result.Reports, report)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatal(err.Error())
	}
	raw = append(raw, '\n')
	if *output == "" {
		_, _ = os.Stdout.Write(raw)
		return
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err.Error())
	}
	if err := os.WriteFile(*output, raw, 0o644); err != nil {
		fatal(err.Error())
	}
}
func fatal(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
