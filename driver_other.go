//go:build !darwin

package main
import _ "modernc.org/sqlite"
const sqliteDriver="sqlite"
