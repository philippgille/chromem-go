# WebAssembly (WASM)

Go can compile to WebAssembly, which you can then use from JavaScript in a Browser or similar environments (Node, Deno, Bun etc.). You could also target WASI (WebAssembly System Interface) and run it in a standalone runtime (wazero, wasmtime, Wasmer), but in this example we focus on the Browser use case.

## How to run

1. Compile the `chromem-go` WASM binding to WebAssembly:
   1. `cd /path/to/chromem-go/wasm`
   2. `GOOS=js GOARCH=wasm go build -o ../examples/webassembly/chromem-go.wasm`
2. Copy Go's wrapper JavaScript:
   1. `cp $(go env GOROOT)/misc/wasm/wasm_exec.js ../examples/webassembly/wasm_exec.js`
3. Serve the files
   1. `cd ../examples/webassembly`
   2. `go run github.com/philippgille/serve@latest -b localhost -p 8080` or similar
4. Open <http://localhost:8080> in your browser
