package configbuild

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zclkkk/qkbox/internal/uriparse"
)

type Mode string

const (
	ModeRule   Mode = "rule"
	ModeGlobal Mode = "global"
	ModeDirect Mode = "direct"
)

type Options struct {
	Listen string
	Port   uint16
	Mode   Mode
}

func DefaultOptions() Options {
	return Options{
		Listen: "127.0.0.1",
		Port:   7890,
		Mode:   ModeRule,
	}
}

func Build(nodes []uriparse.ParsedOutbound, options Options) (string, error) {
	options = normalizeOptions(options)
	if err := validateMode(options.Mode); err != nil {
		return "", err
	}

	outbounds, nodeTags, err := buildOutbounds(nodes)
	if err != nil {
		return "", err
	}
	final := routeFinal(options.Mode, len(nodeTags) > 0)
	config := map[string]any{
		"inbounds": []map[string]any{
			{
				"type":        "mixed",
				"tag":         "mixed-in",
				"listen":      options.Listen,
				"listen_port": int(options.Port),
			},
		},
		"outbounds": outbounds,
		"dns": map[string]any{
			"servers": []map[string]any{
				{"type": "local", "tag": "local"},
			},
		},
		"route": map[string]any{
			"final": final,
		},
	}

	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func normalizeOptions(options Options) Options {
	defaults := DefaultOptions()
	if options.Listen == "" {
		options.Listen = defaults.Listen
	}
	if options.Port == 0 {
		options.Port = defaults.Port
	}
	if options.Mode == "" {
		options.Mode = defaults.Mode
	}
	return options
}

func validateMode(mode Mode) error {
	switch mode {
	case ModeRule, ModeGlobal, ModeDirect:
		return nil
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

func buildOutbounds(nodes []uriparse.ParsedOutbound) ([]map[string]any, []string, error) {
	seen := map[string]struct{}{}
	nodeOutbounds := make([]map[string]any, 0, len(nodes))
	nodeTags := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.Tag == "" {
			return nil, nil, errors.New("node outbound tag is required")
		}
		if node.Type == "" {
			return nil, nil, errors.New("node outbound type is required")
		}
		if _, ok := seen[node.Tag]; ok {
			return nil, nil, fmt.Errorf("duplicate outbound tag: %s", node.Tag)
		}
		copied := cloneOutbound(node.Outbound)
		copied["tag"] = node.Tag
		copied["type"] = node.Type
		nodeOutbounds = append(nodeOutbounds, copied)
		nodeTags = append(nodeTags, node.Tag)
		seen[node.Tag] = struct{}{}
	}

	outbounds := make([]map[string]any, 0, len(nodeOutbounds)+3)
	if len(nodeTags) > 0 {
		outbounds = append(outbounds, map[string]any{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": nodeTags,
			"default":   nodeTags[0],
		})
		seen["proxy"] = struct{}{}
	}
	outbounds = append(outbounds, nodeOutbounds...)
	if _, ok := seen["direct"]; !ok {
		outbounds = append(outbounds, map[string]any{"type": "direct", "tag": "direct"})
	}
	if _, ok := seen["block"]; !ok {
		outbounds = append(outbounds, map[string]any{"type": "block", "tag": "block"})
	}
	return outbounds, nodeTags, nil
}

func routeFinal(mode Mode, hasNodes bool) string {
	if mode == ModeDirect || !hasNodes {
		return "direct"
	}
	return "proxy"
}

func cloneOutbound(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
