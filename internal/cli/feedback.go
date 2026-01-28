package cli

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/constants"
	"github.com/spf13/cobra"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback",
	Short: "Show how to provide feedback about Skulto",
	Long: `Display the feedback URL where you can share your thoughts about Skulto.

Your feedback helps us improve! Report bugs, request features, or just let us know
how Skulto is working for you.`,
	Args: cobra.NoArgs,
	RunE: runFeedback,
}

func runFeedback(cmd *cobra.Command, args []string) error {
	fmt.Println("We'd love to hear from you!")
	fmt.Println()
	fmt.Println("Share your feedback, report bugs, or request features:")
	fmt.Printf("  %s\n", constants.FeedbackURL)
	fmt.Println()
	fmt.Println("Your feedback helps us improve Skulto.")
	return nil
}
