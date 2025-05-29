package core

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/speakeasy-api/openapi/extensions/core"
	"github.com/speakeasy-api/openapi/json"
	"github.com/speakeasy-api/openapi/marshaller"
	"github.com/speakeasy-api/openapi/yml"
	"gopkg.in/yaml.v3"
)

type Arazzo struct {
	Arazzo             marshaller.Node[string]               `key:"arazzo"`
	Info               marshaller.Node[Info]                 `key:"info"`
	SourceDescriptions marshaller.Node[[]*SourceDescription] `key:"sourceDescriptions" required:"true"`
	Workflows          marshaller.Node[[]*Workflow]          `key:"workflows" required:"true"`
	Components         marshaller.Node[*Components]          `key:"components"`
	Extensions         core.Extensions                       `key:"extensions"`

	RootNode *yaml.Node
	Config   *yml.Config
}

func Unmarshal(ctx context.Context, doc io.Reader) (*Arazzo, error) {
	data, err := io.ReadAll(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to read Arazzo document: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("empty document")
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Arazzo document: %w", err)
	}

	var arazzo Arazzo
	if err := marshaller.Unmarshal(ctx, &root, &arazzo); err != nil {
		return nil, err
	}

	arazzo.Config = yml.GetConfigFromDoc(data, &root)

	return &arazzo, nil
}

func (a *Arazzo) Marshal(ctx context.Context, w io.Writer) error {
	cfg := yml.GetConfigFromContext(ctx)

	switch cfg.OutputFormat {
	case yml.OutputFormatYAML:
		enc := yaml.NewEncoder(w)

		enc.SetIndent(cfg.Indentation)
		if err := enc.Encode(a.RootNode); err != nil {
			return err
		}
	case yml.OutputFormatJSON:
		if err := json.YAMLToJSON(a.RootNode, cfg.Indentation, w); err != nil {
			return err
		}
	}

	return nil
}
