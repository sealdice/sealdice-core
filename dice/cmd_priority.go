package dice

import "strings"

// commandExtensionOrder returns activated extensions in command-resolution
// order. Extensions related to the current game system are promoted as one
// stable tier; all relative ordering inside and outside that tier is retained.
func commandExtensionOrder(group *GroupInfo, d *Dice) []*ExtInfo {
	if group == nil || d == nil {
		return nil
	}

	activated := group.GetActivatedExtList(d)
	if group.System == "" || d.GameSystemMap == nil {
		return activated
	}

	tmpl, ok := d.GameSystemMap.Load(group.System)
	if !ok || tmpl == nil || len(tmpl.SetConfig.RelatedExt) == 0 {
		return activated
	}

	preferred := make(map[string]struct{})
	graph := d.activeWithGraph()
	for _, related := range tmpl.SetConfig.RelatedExt {
		related = strings.TrimSpace(related)
		if related == "" {
			continue
		}

		name := d.ExtAliasToName(related)
		preferred[strings.ToLower(name)] = struct{}{}
		for _, chained := range collectChainedNames(d.Logger, graph, name, maxChainDepth) {
			preferred[strings.ToLower(chained)] = struct{}{}
		}
	}
	if len(preferred) == 0 {
		return activated
	}

	ordered := make([]*ExtInfo, 0, len(activated))
	for _, ext := range activated {
		if ext == nil {
			continue
		}
		if _, ok := preferred[strings.ToLower(ext.Name)]; ok {
			ordered = append(ordered, ext)
		}
	}
	for _, ext := range activated {
		if ext == nil {
			continue
		}
		if _, ok := preferred[strings.ToLower(ext.Name)]; !ok {
			ordered = append(ordered, ext)
		}
	}
	return ordered
}
