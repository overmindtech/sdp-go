// This package contains utilities that are used to feed SDP data back and forth
// into an LLM.
//
// The main package contains the objects that are design to be sent over the
// wire, used in memory etc. This package contains the utilities that are used
// to convert between this format and a format that is easy fo an LLM to
// understand, that is more "human" readable.
//
// SInce there are many components that use LLMs within Overmind, these helper
// libraries are centralised
package llm
