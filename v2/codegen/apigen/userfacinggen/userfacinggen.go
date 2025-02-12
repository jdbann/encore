package userfacinggen

import (
	"strings"

	. "github.com/dave/jennifer/jen"

	"encr.dev/pkg/namealloc"
	"encr.dev/pkg/option"
	"encr.dev/v2/app"
	"encr.dev/v2/app/apiframework"
	"encr.dev/v2/codegen"
	"encr.dev/v2/codegen/internal/genutil"
	"encr.dev/v2/internals/resourcepaths"
	"encr.dev/v2/parser/apis/api"
)

// Gen generates the encore.gen.go file containing user-facing
// generated code. If nothing needs to be generated it returns nil.
func Gen(gen *codegen.Generator, svc *app.Service, withSvcStructImpl option.Option[*codegen.VarDecl]) option.Option[*codegen.File] {
	if fw, ok := svc.Framework.Get(); ok {
		f := genUserFacing(gen, fw, withSvcStructImpl)
		return option.Some(f)
	}
	return option.None[*codegen.File]()
}

func genUserFacing(gen *codegen.Generator, svc *apiframework.ServiceDesc, withImpl option.Option[*codegen.VarDecl]) *codegen.File {
	f := gen.InjectFile(svc.RootPkg.ImportPath, svc.RootPkg.Name, svc.RootPkg.FSPath,
		"encore.gen.go", "encoregen")

	f.Jen.HeaderComment("Code generated by encore. DO NOT EDIT.")

	f.Jen.Comment("These functions are automatically generated and maintained by Encore")
	f.Jen.Comment("to simplify calling them from other services, as they were implemented as methods.")
	f.Jen.Comment("They are automatically updated by Encore whenever your API endpoints change.")
	f.Jen.Line()

	for _, ep := range svc.Endpoints {
		if ep.Recv.Empty() {
			continue
		}
		genEndpoint(gen.Util, f, ep, withImpl)
		f.Jen.Line()
	}

	return f
}

func genEndpoint(gu *genutil.Helper, f *codegen.File, ep *api.Endpoint, withImpl option.Option[*codegen.VarDecl]) {
	if ep.Doc != "" {
		for _, line := range strings.Split(strings.TrimSpace(ep.Doc), "\n") {
			f.Jen.Comment(line)
		}
	}

	var pathParamNames []string

	var names namealloc.Allocator
	alloc := func(input string, pathParam bool) string {
		name := names.Get(input)
		if pathParam {
			pathParamNames = append(pathParamNames, name)
		}
		return name
	}

	var (
		ctxName    = alloc("ctx", false)
		rawReqName string
		paramName  string
	)

	f.Jen.Func().Id(ep.Name).ParamsFunc(func(g *Group) {
		g.Id(ctxName).Qual("context", "Context")
		for _, p := range ep.Path.Params() {
			typ := gu.Builtin(p.Pos(), p.ValueType)
			// Wrap wildcards as a slice of values
			if p.Type == resourcepaths.Wildcard {
				typ = Index().Add(typ)
			}
			g.Id(alloc(p.Value, true)).Add(typ)
		}
		if ep.Raw {
			rawReqName = alloc("req", false)
			g.Id(rawReqName).Op("*").Qual("net/http", "Request")
		} else if req := ep.Request; req != nil {
			paramName = alloc("p", false)
			g.Id(paramName).Add(gu.Type(req))
		}
	}).Do(func(s *Statement) {
		if withImpl.Present() {
			if ep.Raw {
				s.Params(Op("*").Qual("net/http", "Response"), Error())
			} else if resp := ep.Response; resp != nil {
				s.Params(gu.Type(resp), Error())
			} else {
				s.Params(Error())
			}
		} else {
			if ep.Raw {
				s.Params(Op("*").Qual("net/http", "Response"), Error())
			} else if resp := ep.Response; resp != nil {
				s.Params(gu.Type(resp), Error())
			} else {
				s.Error()
			}
		}
	}).BlockFunc(func(g *Group) {
		if svcStruct, ok := withImpl.Get(); ok {
			if ep.Raw {
				g.Return(Nil(), Qual("errors", "New").Call(Lit("encore: calling raw endpoints is not yet supported")))
			} else {
				svcName := alloc("svc", false)
				g.List(Id(svcName), Err()).Op(":=").Id(svcStruct.Name()).Dot("Get").Call()
				g.If(Err().Op("!=").Nil()).Block(ReturnFunc(func(g *Group) {
					if ep.Raw {
						g.Nil()
					} else if ep.Response != nil {
						g.Add(gu.Zero(ep.Response))
					}
					g.Err()
				}))

				g.Return(Id("svc").Dot(ep.Name).CallFunc(func(g *Group) {
					g.Id(ctxName)
					for _, name := range pathParamNames {
						g.Id(name)
					}
					if paramName != "" {
						g.Id(paramName)
					}
				}))
			}
		} else {
			g.Comment("The implementation is elided here, and generated at compile-time by Encore.")
			if ep.Raw {
				g.Return(Nil(), Nil())
			} else if ep.Response != nil {
				g.Return(gu.Zero(ep.Response), Nil())
			} else {
				// Just an error return
				g.Return(Nil())
			}
		}
	})
}
