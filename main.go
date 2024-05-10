package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/itchyny/gojq"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
)

type Function struct {
	tfprotov6.Function
	Impl func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError)
}

type FunctionProvider struct {
	ProviderSchema   *tfprotov6.Schema
	StaticFunctions  map[string]*Function
	dynamicFunctions map[string]*Function
	Configure        func(*tfprotov6.DynamicValue) (map[string]*Function, []*tfprotov6.Diagnostic)
}

func (f *FunctionProvider) GetMetadata(context.Context, *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	var functions []tfprotov6.FunctionMetadata
	for name := range f.StaticFunctions {
		functions = append(functions, tfprotov6.FunctionMetadata{Name: name})
	}

	return &tfprotov6.GetMetadataResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Functions:          functions,
	}, nil
}
func (f *FunctionProvider) GetProviderSchema(context.Context, *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	functions := make(map[string]*tfprotov6.Function)
	for name, fn := range f.StaticFunctions {
		functions[name] = &fn.Function
	}

	return &tfprotov6.GetProviderSchemaResponse{
		ServerCapabilities: &tfprotov6.ServerCapabilities{GetProviderSchemaOptional: true},
		Provider:           f.ProviderSchema,
		Functions:          functions,
	}, nil
}
func (f *FunctionProvider) ValidateProviderConfig(ctx context.Context, req *tfprotov6.ValidateProviderConfigRequest) (*tfprotov6.ValidateProviderConfigResponse, error) {
	// Passthrough
	return &tfprotov6.ValidateProviderConfigResponse{PreparedConfig: req.Config}, nil
}
func (f *FunctionProvider) ConfigureProvider(ctx context.Context, req *tfprotov6.ConfigureProviderRequest) (*tfprotov6.ConfigureProviderResponse, error) {
	funcs, diags := f.Configure(req.Config)
	f.dynamicFunctions = funcs
	return &tfprotov6.ConfigureProviderResponse{
		Diagnostics: diags,
	}, nil
}
func (f *FunctionProvider) StopProvider(context.Context, *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	return &tfprotov6.StopProviderResponse{}, nil
}
func (f *FunctionProvider) ValidateResourceConfig(context.Context, *tfprotov6.ValidateResourceConfigRequest) (*tfprotov6.ValidateResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) UpgradeResourceState(context.Context, *tfprotov6.UpgradeResourceStateRequest) (*tfprotov6.UpgradeResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadResource(context.Context, *tfprotov6.ReadResourceRequest) (*tfprotov6.ReadResourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) PlanResourceChange(context.Context, *tfprotov6.PlanResourceChangeRequest) (*tfprotov6.PlanResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ApplyResourceChange(context.Context, *tfprotov6.ApplyResourceChangeRequest) (*tfprotov6.ApplyResourceChangeResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ImportResourceState(context.Context, *tfprotov6.ImportResourceStateRequest) (*tfprotov6.ImportResourceStateResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ValidateDataResourceConfig(context.Context, *tfprotov6.ValidateDataResourceConfigRequest) (*tfprotov6.ValidateDataResourceConfigResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) ReadDataSource(context.Context, *tfprotov6.ReadDataSourceRequest) (*tfprotov6.ReadDataSourceResponse, error) {
	return nil, errors.New("not supported")
}
func (f *FunctionProvider) CallFunction(ctx context.Context, req *tfprotov6.CallFunctionRequest) (*tfprotov6.CallFunctionResponse, error) {
	if fn, ok := f.StaticFunctions[req.Name]; ok {
		ret, err := fn.Impl(req.Arguments)
		return &tfprotov6.CallFunctionResponse{
			Result: ret,
			Error:  err,
		}, nil
	}
	if f.dynamicFunctions != nil {
		if fn, ok := f.dynamicFunctions[req.Name]; ok {
			ret, err := fn.Impl(req.Arguments)
			return &tfprotov6.CallFunctionResponse{
				Result: ret,
				Error:  err,
			}, nil
		}
	}
	return nil, errors.New("unknown function " + req.Name)
}
func (f *FunctionProvider) GetFunctions(context.Context, *tfprotov6.GetFunctionsRequest) (*tfprotov6.GetFunctionsResponse, error) {
	functions := make(map[string]*tfprotov6.Function)
	for name, fn := range f.StaticFunctions {
		functions[name] = &fn.Function
	}
	for name, fn := range f.dynamicFunctions {
		functions[name] = &fn.Function
	}

	return &tfprotov6.GetFunctionsResponse{
		Functions: functions,
	}, nil
}

func main() {
	err := tf6server.Serve("registry.opentofu.org/opentofu/jq", func() tfprotov6.ProviderServer {
		provider := &FunctionProvider{
			ProviderSchema: &tfprotov6.Schema{
				Block: &tfprotov6.SchemaBlock{
					Attributes: []*tfprotov6.SchemaAttribute{
						&tfprotov6.SchemaAttribute{
							Name:     "jq",
							Type:     tftypes.String,
							Required: false,
						},
					},
				},
			},
			Configure: func(config *tfprotov6.DynamicValue) (map[string]*Function, []*tfprotov6.Diagnostic) {
				res, err := config.Unmarshal(tftypes.Map{ElementType: tftypes.String})
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}
				cfg := make(map[string]tftypes.Value)
				err = res.As(&cfg)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}

				codeVal := cfg["jq"]
				var code string
				err = codeVal.As(&code)
				if err != nil {
					return nil, []*tfprotov6.Diagnostic{&tfprotov6.Diagnostic{
						Severity: tfprotov6.DiagnosticSeverityError,
						Summary:  "Invalid configure payload",
						Detail:   err.Error(),
					}}
				}

				functions := make(map[string]*Function)

				return functions, nil
			},
			StaticFunctions: map[string]*Function{
				"exec": &Function{
					tfprotov6.Function{
						Parameters: []*tfprotov6.FunctionParameter{&tfprotov6.FunctionParameter{
							Name: "code",
							Type: tftypes.String,
						}},
						VariadicParameter: &tfprotov6.FunctionParameter{
							AllowNullValue: true,
							Name:           "args",
							Type:           tftypes.DynamicPseudoType,
						},
						Return: &tfprotov6.FunctionReturn{
							Type: tftypes.DynamicPseudoType,
						},
					},
					func(args []*tfprotov6.DynamicValue) (*tfprotov6.DynamicValue, *tfprotov6.FunctionError) {
						codeVal, err := args[0].Unmarshal(tftypes.String)
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						var code string
						err = codeVal.As(&code)
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}
						args = args[1:]

						query, err := gojq.Parse(code)
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						parsedArg, err := parseArg(args[0])
						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						iter := query.Run(parsedArg)

						var store []interface{}

						for {
							v, ok := iter.Next()
							if !ok {
								break
							}
							if err, ok := v.(error); ok {
								if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
									break
								}
								return nil, &tfprotov6.FunctionError{Text: err.Error()}
							}

							store = append(store, v)
						}

						output, err := processOutput(store)

						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						result, err := msgpack.Marshal(cty.StringVal(string(output)), cty.DynamicPseudoType)

						if err != nil {
							return nil, &tfprotov6.FunctionError{Text: err.Error()}
						}

						return &tfprotov6.DynamicValue{
							MsgPack: result,
						}, nil
					},
				},
			},
		}
		return provider
	})
	if err != nil {
		panic(err)
	}
}

func parseArg(arg *tfprotov6.DynamicValue) (map[string]interface{}, error) {
	parsedInput := make(map[string]interface{})
	v, err := msgpack.Unmarshal(arg.MsgPack, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("problem parsing the argument")
	}
	json.Unmarshal([]byte(v.AsString()), &parsedInput)
	return parsedInput, err
}

func processOutput(store []interface{}) ([]byte, error) {
	if len(store) == 1 {
		output, err := json.Marshal(store[0])
		return output, err
	} else {
		output, err := json.Marshal(store)
		return output, err
	}

}
