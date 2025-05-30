package format

import (
	"os"
	"log"
	"fmt"
)

// Codes pour le terminal
var rouge string = "\033[1;31m"
var orange string = "\033[1;33m"
var raz string = "\033[0;00m"
var green string = "\033[0;32m"
var pid = os.Getpid()
var stderr = log.New(os.Stderr, "", 0)

func Format_d(where string, who string, what string) string {
	return fmt.Sprintf("%s + [%12.12s %d] %-15.15s : %s\n",raz, who, pid, where, what)
}
func Format_w(where string, who string, what string) string {
	return fmt.Sprintf("%s * [%12.12s %d] %-15.15s : %s\n%s", orange, who, pid, where, what, raz)
}
func Format_e(where string, who string, what string) string {
    return fmt.Sprintf("%s ! [%12.12s %d] %.15s : %s\n%s", rouge, who, pid, where, what, raz)
}
func Format_g(where string, who string, what string) string {
    return fmt.Sprintf("%s ! [%12.12s %d] %.15s : %s\n%s", green, who, pid, where, what, raz)
}
func Display(message string) {
	stderr.Printf("%s\n", message)
}



