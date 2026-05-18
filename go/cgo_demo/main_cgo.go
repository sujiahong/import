package main

//#include "hello.h"
import "C"

func main() {
	//println("hello cgo")
	// C.puts(C.CString("Hello World\n"))
	C.SayHello(C.CString("helol world\n"))
}
