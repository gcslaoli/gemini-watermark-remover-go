// Package watermark provides a lossless Gemini watermark remover implemented in Go.
//
// It ports the reverse alpha blending algorithm from the original JavaScript
// project and ships with embedded watermark alpha maps for the 48x48 and 96x96
// logos used by Gemini. The package works entirely in memory; no network or GPU
// is required.
package watermark
