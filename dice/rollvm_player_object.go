package dice

import (
	"fmt"
	"sort"

	ds "github.com/sealdice/dicescript"
)

func actorObjectMethod(name string, fn ds.NativeFunctionDef) *ds.VMValue {
	return ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name:       name,
		Params:     []string{},
		Defaults:   nil,
		NativeFunc: fn,
	})
}

func bindActorObjectMethod(object *ds.VMValue, method *ds.VMValue) *ds.VMValue {
	if method == nil {
		return nil
	}
	fd, ok := method.ReadNativeFunctionData()
	if !ok {
		return method
	}
	cloned := *fd
	cloned.Self = object.Clone()
	return ds.NewNativeFunctionVal(&cloned)
}

func resolveActorAttrName(ctx *MsgContext, name string) string {
	if ctx != nil && ctx.SystemTemplate != nil {
		return ctx.SystemTemplate.GetAlias(name)
	}
	return name
}

func actorAttrExistsWithTemplateDefault(ctx *MsgContext, canonicalName string) (*ds.VMValue, bool) {
	if ctx == nil || ctx.SystemTemplate == nil {
		return nil, false
	}

	ctx2 := ctx.ShallowCopy()
	ctx2.vm = nil
	ctx2.CreateVmIfNotExists()
	if parentVM := ctx.vm; parentVM != nil {
		ctx2.vm.UpCtx = parentVM
		ctx2.vm.Attrs = parentVM.Attrs
		ctx2.vm.Config = parentVM.Config
	}

	if v, _, _, exists := ctx.SystemTemplate.GetDefaultValueEx0(ctx2, canonicalName); exists {
		return v, true
	}
	return nil, false
}

func loadActorAttrValue(ctx *MsgContext, canonicalName string) (*ds.VMValue, bool) {
	if ctx == nil || ctx.Dice == nil || ctx.Player == nil {
		return nil, false
	}

	attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx)
	if err == nil && attrs != nil {
		if ctx.SystemTemplate != nil {
			ctx.syncAttrsForTemplate(attrs, ctx.SystemTemplate.GameSystemTemplateV2)
		}
		if v, exists := attrs.LoadX(canonicalName); exists {
			return v, true
		}
	}

	return actorAttrExistsWithTemplateDefault(ctx, canonicalName)
}

func actorVisibleKeys(ctx *MsgContext) []string {
	keys := map[string]struct{}{}

	if ctx != nil && ctx.Dice != nil && ctx.Player != nil {
		if attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx); err == nil && attrs != nil {
			if ctx.SystemTemplate != nil {
				ctx.syncAttrsForTemplate(attrs, ctx.SystemTemplate.GameSystemTemplateV2)
			}
			attrs.Range(func(key string, _ *ds.VMValue) bool {
				keys[key] = struct{}{}
				return true
			})
		}
	}

	if ctx != nil {
		if tmpl := ctx.SystemTemplate; tmpl != nil && tmpl.GameSystemTemplateV2 != nil {
			for key := range tmpl.Attrs.Defaults {
				keys[key] = struct{}{}
			}
			for key := range tmpl.Attrs.DefaultsComputed {
				keys[key] = struct{}{}
			}
		}
	}

	delete(keys, attrsTemplateVersionKey)

	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func newActorNativeObject(ctx *MsgContext, objectName string) *ds.VMValue {
	visibleKeys := func() []string {
		return actorVisibleKeys(ctx)
	}
	resolveVisibleValue := func(vm *ds.Context, key string, isRaw bool) *ds.VMValue {
		val, exists := loadActorAttrValue(ctx, key)
		if !exists || val == nil {
			return nil
		}
		if !isRaw && val.TypeId == ds.VMTypeComputedValue {
			val = val.ComputedExecute(vm, &ds.BufferSpan{})
			if vm.Error != nil {
				return nil
			}
		}
		return val
	}
	readVisibleValue := func(key string) *ds.VMValue {
		val, exists := loadActorAttrValue(ctx, key)
		if !exists || val == nil {
			return ds.NewIntVal(0)
		}
		return val
	}
	methods := map[string]*ds.VMValue{
		"keys": actorObjectMethod(objectName+".keys", func(vm *ds.Context, this *ds.VMValue, _ []*ds.VMValue) *ds.VMValue {
			keys := visibleKeys()
			out := make([]*ds.VMValue, 0, len(keys))
			for _, key := range keys {
				out = append(out, ds.NewStrVal(key))
			}
			return ds.NewArrayValRaw(out)
		}),
		"values": actorObjectMethod(objectName+".values", func(vm *ds.Context, this *ds.VMValue, _ []*ds.VMValue) *ds.VMValue {
			keys := visibleKeys()
			values := make([]*ds.VMValue, 0, len(keys))
			for _, key := range keys {
				values = append(values, readVisibleValue(key))
			}
			return ds.NewArrayValRaw(values)
		}),
		"items": actorObjectMethod(objectName+".items", func(vm *ds.Context, this *ds.VMValue, _ []*ds.VMValue) *ds.VMValue {
			keys := visibleKeys()
			items := make([]*ds.VMValue, 0, len(keys))
			for _, key := range keys {
				items = append(items, ds.NewArrayVal(ds.NewStrVal(key), readVisibleValue(key)))
			}
			return ds.NewArrayValRaw(items)
		}),
		"len": actorObjectMethod(objectName+".len", func(vm *ds.Context, this *ds.VMValue, _ []*ds.VMValue) *ds.VMValue {
			return ds.NewIntVal(ds.IntType(len(visibleKeys())))
		}),
		"has": ds.NewNativeFunctionVal(&ds.NativeFunctionData{
			Name:     objectName + ".has",
			Params:   []string{"key"},
			Defaults: nil,
			NativeFunc: func(vm *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
				key, err := params[0].AsDictKey()
				if err != nil {
					vm.Error = err
					return nil
				}
				canonicalName := resolveActorAttrName(ctx, key)
				if _, exists := loadActorAttrValue(ctx, canonicalName); exists {
					return ds.NewIntVal(1)
				}
				return ds.NewIntVal(0)
			},
		}),
		"get": ds.NewNativeFunctionVal(&ds.NativeFunctionData{
			Name:     objectName + ".get",
			Params:   []string{"key", "default"},
			Defaults: []*ds.VMValue{nil, ds.NewNullVal()},
			NativeFunc: func(vm *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
				key, err := params[0].AsDictKey()
				if err != nil {
					vm.Error = err
					return nil
				}
				canonicalName := resolveActorAttrName(ctx, key)
				if val := resolveVisibleValue(vm, canonicalName, false); val != nil {
					return val
				}
				if len(params) > 1 {
					return params[1]
				}
				return ds.NewNullVal()
			},
		}),
		"getRaw": ds.NewNativeFunctionVal(&ds.NativeFunctionData{
			Name:     objectName + ".getRaw",
			Params:   []string{"key", "default"},
			Defaults: []*ds.VMValue{nil, ds.NewNullVal()},
			NativeFunc: func(vm *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
				key, err := params[0].AsDictKey()
				if err != nil {
					vm.Error = err
					return nil
				}
				canonicalName := resolveActorAttrName(ctx, key)
				if val := resolveVisibleValue(vm, canonicalName, true); val != nil {
					return val
				}
				if len(params) > 1 {
					return params[1]
				}
				return ds.NewNullVal()
			},
		}),
	}

	var object *ds.VMValue
	object = ds.NewNativeObjectVal(&ds.NativeObjectData{
		Name: objectName,
		AttrSet: func(vm *ds.Context, name string, v *ds.VMValue) {
			if ctx == nil || ctx.Dice == nil || ctx.Player == nil {
				vm.Error = fmt.Errorf("%s is unavailable in current context", objectName)
				return
			}
			canonicalName := resolveActorAttrName(ctx, name)
			attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx)
			if err != nil {
				vm.Error = err
				return
			}
			attrs.Store(canonicalName, v.Clone())
		},
		AttrGet: func(vm *ds.Context, name string) *ds.VMValue {
			if method, ok := methods[name]; ok {
				return bindActorObjectMethod(object, method)
			}
			canonicalName := resolveActorAttrName(ctx, name)
			if v, exists := loadActorAttrValue(ctx, canonicalName); exists {
				return v
			}
			return ds.NewIntVal(0)
		},
		ItemSet: func(vm *ds.Context, index *ds.VMValue, v *ds.VMValue) {
			if ctx == nil || ctx.Dice == nil || ctx.Player == nil {
				vm.Error = fmt.Errorf("%s is unavailable in current context", objectName)
				return
			}
			key, err := index.AsDictKey()
			if err != nil {
				vm.Error = err
				return
			}
			canonicalName := resolveActorAttrName(ctx, key)
			attrs, err := ctx.Dice.AttrsManager.LoadByCtx(ctx)
			if err != nil {
				vm.Error = err
				return
			}
			attrs.Store(canonicalName, v.Clone())
		},
		ItemGet: func(vm *ds.Context, index *ds.VMValue) *ds.VMValue {
			key, err := index.AsDictKey()
			if err != nil {
				vm.Error = err
				return nil
			}
			canonicalName := resolveActorAttrName(ctx, key)
			if v, exists := loadActorAttrValue(ctx, canonicalName); exists {
				return v
			}
			return ds.NewIntVal(0)
		},
		DirFunc: func(vm *ds.Context) []*ds.VMValue {
			out := make([]*ds.VMValue, 0, len(methods))
			for key := range methods {
				out = append(out, ds.NewStrVal(key))
			}
			sort.Slice(out, func(i, j int) bool {
				return out[i].ToString() < out[j].ToString()
			})
			return out
		},
		ToString: func(vm *ds.Context) string {
			return fmt.Sprintf("nobject %s", objectName)
		},
	})
	return object
}
