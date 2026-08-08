package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// emitAttribution tells the user which cached identity produced the listing.
//
// Discovery output is meaningless without knowing who asked, since two
// identities in the same cache see different tenants. It goes to the error
// stream so that stdout stays exactly the payload a script parses: under
// --json, a bare array and nothing else.
//
// An empty username means no identity was resolved, and inventing a line for it
// would be noise.
func emitAttribution(cmd *cobra.Command, username string) {
	if username == "" {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Enumerating as %s\n", username)
}
