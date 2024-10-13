// colors.go
package main

import "github.com/fatih/color"

// Define color variables
var (
    HeaderColor   = color.New(color.FgCyan, color.Bold)
    SuccessColor  = color.New(color.FgGreen)
    ErrorColor    = color.New(color.FgRed, color.Bold)
    WarningColor  = color.New(color.FgYellow)
    InfoColor     = color.New(color.FgBlue)
    PromptColor   = color.New(color.FgMagenta)
    ResetColor    = color.New(color.Reset)
)