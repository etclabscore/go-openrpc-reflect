### Logical flow:

- settings get set (defaults CAN BE selected; otherwise settings for standard library instantiate)

- root-level fields (__besides `.methods` and its children (`.contentDescriptors` and `.schemas`)__) get set
  for the document, eg. `.info`, `.servers`, `.externalDocs`, &c.
  
  > Note that `.contentDescriptors` will never be filled. `.methods.#.contentDescriptors.#.schemas` will be "flattened"
  > to `.schemas`, but `.methods.#.contentDescriptors` will not be flattened to `.contentDescriptors`.
  > 
  > ... __Maybe the document should be left completely inline.__ Probably maybe, even.                                                                                                                                                           

- receivers get registered, optionally with a "name," which is really an override for _receiver name_ (eg. `FakeMath.Add` -> `namedmath.Add`)

register receiver `func Register(<name>, receiver interface{})` =>

- for all of a receiver's methods:

    - if !`methodIsEligible = func(r, f reflect.Value) bool`; continue
    
      > `methodsIsEligible` reiterates the eligibility function of the in-use RPC package, eg.  
      >  - go standard rpc package `func(r *Receiver, firstParam ArgType, secondParam *ReplyType) error`
      >    - `func StandardMethodIsEligible(r, f reflect.Value) bool`
      >  - ethereum/go-ethereum/rpc package `func(r *Receiver, <anyOf <[*|]ArgTypes...>) (<anyOf [*|]ResultType>, <anyOf error>)`
      >    - `func EthereumMethodIsEligible(r, f reflect.Value) bool`
      >  - ... which can be extended as:iiiiiiiiiiiiiiiii/
    
    ```go
    func myAppMethodIsEligible(r, f reflect.Value) bool {
        // Safely using .In(0) and .Out(0) assuming we know that, say, EthereumMethod demands at least 1 argument 
        // (which it actually doesn't, but just for example and to keep the example short).    
        return go_openrpc_reflect.EthereumMethodIsEligible(r, f) && !isContextType(f.In(0)) && !isSubscriptionType(f.Out(0))             
    }
    ```
      
    - `methods=append(methods,` @@method @ code`)`

after register receiver, then =>

- (?:under consideration) flatten methods' content descriptors' schemas to the document `.schemas` object


method @ code: `getMethod(r, f reflect.Value) (goopenrpcT.Method, error)`

- `getMethodName(r, f reflect.Value) (string, error)`
- `getMethodTags(r, f reflect.Value, *ast.FuncDecl) ([]string, error)` 
- `getMethodSummary(r, f reflect.Value, *ast.FuncDecl) (string, error)` 
- `getMethodDescription(r, f reflect.Value, *ast.FuncDecl) (string, error)` 
- `getMethodArgs(r, f reflect.Value, *ast.FuncDecl) ([]*goopenrpcT.ContentDescriptor, error)`
- `getMethodReply(r, f reflect.Value, *ast.FuncDecl) (*goopenrpcT.ContentDescriptor, error)`
- `getMethodExternalDocs`  ...
- `getMethodLinks` ...
- `getMethodServers` ...

> Get content descriptors for __all__ of the methods fields. These can be subsetted into args and replies by the method.
 
get content descriptors @ code: `getAllContentDescriptors(r, f reflect.Value, *ast.FuncDecl) ([]*goopenrpcT.ContentDescriptor, error)`

- for all of a method's fields

  `cd=`@@content descriptor @ code
  
  if `cd == nil`; continue
  
  `cds=append(cds, cd)`

content descriptor @ code`func getContentDescriptor(r, f reflect.Value, field *ast.FieldDecl) (*goopenrpcT.ContentDescriptor, error)`

> Run the schema reflection first, making it's value available for reference or use by
the content descriptor methods.

- `sch=`@@schema @ code

> We have the same parameters (runtime and static information) as would the schema step. 
The schema step will not have an eligibility/skip feature; if the field (the content descriptor) should be skipped, skip now. 

- `getContentDescriptorName(r, f reflect.Value, decl *ast.FieldDecl) (string, error)`
- `getContentDescriptorSummary(r, f reflect.Value, *ast.FieldDecl) (string, error)` 
- `getContentDescriptorDescription(r, f reflect.Value, *ast.FieldDecl) (string, error)`
- `getContentDescriptorSchema(r, f reflect.Value, *ast.FieldDecl) (spec.Schema, error)`
- ...

schema @ code

- `jsonschema.Reflector.TypeMapper: func(ty reflect.Type) *jsonschema.Type`
- `onSchema(ty reflect.Type) []func (*spec.Schema) error`
  > The functions returned in a slice will be passed to a depth-first mutable walk of the reflected schema in order.


