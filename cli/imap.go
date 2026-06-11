package cli

import (
	"fmt"
	"os"

	"github.com/emersion/go-imap"
	"github.com/spf13/cobra"
	"github.com/wryfi/shemail/imaputils"
	"github.com/wryfi/shemail/util"
	"golang.org/x/term"
)

// ListFolders generates a command to print a list of imap folders on terminal
func ListFolders() *cobra.Command {
	var (
		long  bool
		dates bool
	)
	cmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"folders"},
		Short:   "print a list of folders in the configured mailbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)

			if long || dates {
				folders, err := imaputils.ListFoldersWithStatus(imaputils.SheDialer, account, dates)
				if err != nil {
					return fmt.Errorf("Error listing folders: %w", err)
				}
				util.TabulateFolders(folders, dates).Render()
				return nil
			}

			folders, err := imaputils.ListFolders(imaputils.SheDialer, account)
			if err != nil {
				return fmt.Errorf("Error listing folders: %w", err)
			}

			for _, folder := range folders {
				fmt.Println(folder)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&long, "long", "l", false, "show message and unread counts per folder")
	cmd.Flags().BoolVar(&dates, "dates", false, "also show each folder's message date range (slower; implies -l)")
	return cmd
}

// SearchFolder generates a command to search a folder for messages based on various criteria
func SearchFolder() *cobra.Command {
	var (
		endDate      string
		from         string
		or           bool
		startDate    string
		subject      []string
		to           string
		notFrom      string
		notSubject   []string
		notTo        string
		unread       bool
		read         bool
		moveTo       string
		deleteFrom   bool
		purge        bool
		largerThan   string
		smallerThan  string
		subjectRegex bool
		markRead     bool
		markUnread   bool
		sortBy       string
		reverse      bool
		copyTo       string
		countOnly    bool
		assumeYes    bool
	)
	cmd := &cobra.Command{
		Use:     "find <folder>",
		Short:   "search the specified folder for messages",
		Aliases: []string{"search"},
		Args:    validateFolderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)
			searchOpts, err := buildSearchOptions(to, from, subject, notTo, notFrom, notSubject, startDate, endDate, largerThan, smallerThan, read, unread)
			if err != nil {
				return fmt.Errorf("error building search options: %v", err)
			}
			searchOpts.SubjectRegex = subjectRegex

			sortField, err := imaputils.ParseSortField(sortBy)
			if err != nil {
				return err
			}

			var criteria *imap.SearchCriteria
			if or {
				criteria = imaputils.BuildORSearchCriteria(searchOpts)
			} else {
				criteria = imaputils.BuildSearchCriteria(searchOpts)
			}

			messages, err := imaputils.SearchMessages(imaputils.SheDialer, account, args[0], criteria)
			if err != nil {
				return fmt.Errorf("error searching folder %s: %w", args[0], err)
			}

			// Subject matching is performed client-side against the decoded
			// subject: server-side SEARCH SUBJECT is backed by a full-text index
			// and is unreliable, especially for negation.
			messages, err = imaputils.FilterBySubject(messages, searchOpts)
			if err != nil {
				return fmt.Errorf("error filtering by subject: %w", err)
			}

			imaputils.SortMessages(messages, sortField, reverse)

			if countOnly {
				fmt.Println(len(messages))
				return nil
			}

			// Determine which action was requested (the flags are mutually
			// exclusive) and label it for the picker. --purge upgrades delete to
			// a permanent expunge for this run.
			account.Purge = account.Purge || purge
			actionLabel := ""
			switch {
			case moveTo != "":
				actionLabel = "move to " + moveTo
			case copyTo != "":
				actionLabel = "copy to " + copyTo
			case markRead:
				actionLabel = "mark as read"
			case markUnread:
				actionLabel = "mark as unread"
			case deleteFrom:
				actionLabel = "delete"
				if account.Purge {
					actionLabel = "permanently delete"
				}
			}

			// A bare listing, or any action with --yes, prints the static table.
			// The interactive picker renders its own table, so skip the static
			// print when it will run.
			if actionLabel == "" || assumeYes {
				rendered, err := util.RenderMessages(messages)
				if err != nil {
					return fmt.Errorf("error rendering messages: %w", err)
				}
				fmt.Println(rendered)
			}
			if actionLabel == "" {
				return nil
			}

			targets, proceed, err := resolveActionTargets(messages, actionLabel, assumeYes)
			if err != nil {
				return err
			}
			if !proceed {
				return nil
			}

			switch {
			case moveTo != "":
				if err := imaputils.MoveMessages(imaputils.SheDialer, account, targets, args[0], moveTo, 100); err != nil {
					return fmt.Errorf("failed to move messages to %s: %w", moveTo, err)
				}
			case copyTo != "":
				if err := imaputils.CopyMessages(imaputils.SheDialer, account, targets, args[0], copyTo); err != nil {
					return fmt.Errorf("failed to copy messages to %s: %w", copyTo, err)
				}
			case markRead || markUnread:
				if err := imaputils.MarkMessages(imaputils.SheDialer, account, targets, args[0], markRead); err != nil {
					state := "read"
					if markUnread {
						state = "unread"
					}
					return fmt.Errorf("failed to mark messages as %s: %w", state, err)
				}
			case deleteFrom:
				if err := imaputils.DeleteMessages(imaputils.SheDialer, account, targets, args[0]); err != nil {
					return fmt.Errorf("failed to delete messages from %s: %w", args[0], err)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&to, "to", "t", "", "find messages to this address")
	cmd.Flags().StringVarP(&from, "from", "f", "", "find messages from this address")
	cmd.Flags().StringArrayVarP(&subject, "subject", "s", nil, "match subject (repeatable; matches if any matches)")
	cmd.Flags().StringVar(&notTo, "not-to", "", "exclude messages to this address")
	cmd.Flags().StringVar(&notFrom, "not-from", "", "exclude messages from this address")
	cmd.Flags().StringArrayVar(&notSubject, "not-subject", nil, "exclude messages whose subject matches (repeatable; excludes if any matches)")
	cmd.Flags().BoolVar(&subjectRegex, "subject-regex", false, "treat --subject and --not-subject as regular expressions")
	cmd.Flags().StringVarP(&startDate, "after", "a", "", "find messages received after date (format: `2006-01-02`)")
	cmd.Flags().StringVarP(&endDate, "before", "b", "", "find messages received before date (format: `2006-01-02`)")
	cmd.Flags().StringVar(&largerThan, "larger-than", "", "find messages larger than this size (e.g. 500K, 10M)")
	cmd.Flags().StringVar(&smallerThan, "smaller-than", "", "find messages smaller than this size (e.g. 500K, 10M)")
	cmd.Flags().BoolVarP(&unread, "unread", "u", false, "find only unread messages")
	cmd.Flags().BoolVarP(&read, "read", "r", false, "find only read messages")
	cmd.Flags().BoolVarP(&or, "or", "o", false, "OR search criteria instead of AND")
	cmd.Flags().StringVarP(&moveTo, "move", "m", "", "move messages to <folder>")
	cmd.Flags().StringVar(&copyTo, "copy", "", "copy messages to <folder>")
	cmd.Flags().BoolVarP(&deleteFrom, "delete", "d", false, "delete messages")
	cmd.Flags().BoolVarP(&purge, "purge", "p", false, "with --delete, permanently expunge messages instead of moving them to trash")
	cmd.Flags().BoolVar(&markRead, "mark-read", false, "mark messages as read (\\Seen)")
	cmd.Flags().BoolVar(&markUnread, "mark-unread", false, "mark messages as unread")
	cmd.Flags().StringVar(&sortBy, "sort", "date", "sort by: date, subject, from, to, size, unread")
	cmd.Flags().BoolVarP(&reverse, "reverse", "R", false, "reverse the sort order")
	cmd.Flags().BoolVar(&countOnly, "count", false, "print only the number of matching messages")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "skip confirmation prompts (assume yes)")
	// --read and --unread are contradictory: requiring both Seen and not-Seen
	// matches nothing. Reject the combination up front instead of silently
	// returning zero results.
	cmd.MarkFlagsMutuallyExclusive("read", "unread")
	// At most one action per run. Combining them is either nonsensical (move
	// then delete the same UIDs from a folder they left) or ambiguous in
	// ordering; run separate passes if you want more than one.
	cmd.MarkFlagsMutuallyExclusive("move", "copy", "delete", "mark-read", "mark-unread", "count")
	return cmd
}

// isInteractive reports whether stdin is a terminal, i.e. whether we can prompt
// the user (run the picker) rather than refusing or hanging.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// resolveActionTargets decides which messages an action applies to. With --yes
// it acts on all of them. Otherwise, at a terminal it launches the picker so the
// user can deselect messages before acting; in a non-interactive session it
// refuses rather than act blindly. proceed is false when the caller should stop
// without acting (the user cancelled, selected nothing, or an error occurred).
func resolveActionTargets(messages []*imap.Message, actionLabel string, assumeYes bool) (targets []*imap.Message, proceed bool, err error) {
	if assumeYes {
		return messages, true, nil
	}
	if !isInteractive() {
		return nil, false, fmt.Errorf("refusing to %s %d messages without --yes in a non-interactive session", actionLabel, len(messages))
	}
	kept, committed, err := util.SelectMessages(messages, actionLabel)
	if err != nil {
		return nil, false, err
	}
	if !committed {
		fmt.Println("operation cancelled")
		return nil, false, nil
	}
	if len(kept) == 0 {
		fmt.Println("no messages selected; nothing to do")
		return nil, false, nil
	}
	return kept, true, nil
}

// CountMessagesBySender generates a command to list all the senders represented mailbox by how many messages they sent
func CountMessagesBySender() *cobra.Command {
	var (
		threshold int
		after     string
		before    string
	)
	cmd := &cobra.Command{
		Use:   "senders <folder>",
		Short: "print a list of senders in the configured mailbox",
		Args:  validateFolderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)

			startDate, err := parseOptionalDate(after)
			if err != nil {
				return fmt.Errorf("error parsing after date %s: %w", after, err)
			}
			endDate, err := parseOptionalDate(before)
			if err != nil {
				return fmt.Errorf("error parsing before date %s: %w", before, err)
			}

			data, err := imaputils.CountMessagesBySender(imaputils.SheDialer, account, args[0], threshold, startDate, endDate)
			if err != nil {
				return fmt.Errorf("error counting messages: %w", err)
			}
			table := util.TabulateSenders(data)
			table.Render()
			return nil
		},
	}
	cmd.Flags().IntVarP(&threshold, "threshold", "t", 1, "only show senders with at least this many messages")
	cmd.Flags().StringVarP(&after, "after", "a", "", "only count messages received after date (format: `2006-01-02`)")
	cmd.Flags().StringVarP(&before, "before", "b", "", "only count messages received before date (format: `2006-01-02`)")
	return cmd
}

// EmptyTrash generates a command to permanently delete all messages in the
// account's trash folder.
func EmptyTrash() *cobra.Command {
	var assumeYes bool
	cmd := &cobra.Command{
		Use:   "empty-trash",
		Short: "permanently delete all messages in the trash folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)

			folder, err := imaputils.FindTrashFolder(imaputils.SheDialer, account)
			if err != nil {
				return fmt.Errorf("failed to find trash folder: %w", err)
			}

			count, err := imaputils.FolderMessageCount(imaputils.SheDialer, account, folder)
			if err != nil {
				return fmt.Errorf("failed to count messages in %s: %w", folder, err)
			}

			if !assumeYes && !util.GetConfirmation(fmt.Sprintf("really permanently delete all %d messages in %q?", count, folder)) {
				fmt.Println("operation cancelled")
				return nil
			}

			deleted, err := imaputils.EmptyFolder(imaputils.SheDialer, account, folder)
			if err != nil {
				return fmt.Errorf("failed to empty %s: %w", folder, err)
			}
			fmt.Printf("permanently deleted %d messages from %s\n", deleted, folder)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

// Dedupe generates a command to delete duplicate messages (sharing a
// Message-ID) within a folder, keeping the oldest copy of each.
func Dedupe() *cobra.Command {
	var (
		purge     bool
		assumeYes bool
	)
	cmd := &cobra.Command{
		Use:   "dedupe <folder>",
		Short: "delete duplicate messages in a folder, keeping the oldest copy",
		Args:  validateFolderArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)

			criteria := imaputils.BuildSearchCriteria(imaputils.SearchOptions{})
			messages, err := imaputils.SearchMessages(imaputils.SheDialer, account, args[0], criteria)
			if err != nil {
				return fmt.Errorf("error searching folder %s: %w", args[0], err)
			}

			// Keep the oldest copy of each message: sort oldest-first so the
			// first occurrence of each Message-ID (the one kept) is the oldest.
			imaputils.SortMessages(messages, imaputils.SortDate, true)
			duplicates := imaputils.FindDuplicates(messages)

			if len(duplicates) == 0 {
				fmt.Printf("no duplicate messages found in %s\n", args[0])
				return nil
			}

			rendered, err := util.RenderMessages(duplicates)
			if err != nil {
				return fmt.Errorf("error rendering messages: %w", err)
			}
			fmt.Println(rendered)

			account.Purge = account.Purge || purge
			action := "delete"
			if account.Purge {
				action = "permanently delete"
			}
			if assumeYes || util.GetConfirmation(fmt.Sprintf("really %s %d duplicate messages from %s?", action, len(duplicates), args[0])) {
				if err := imaputils.DeleteMessages(imaputils.SheDialer, account, duplicates, args[0]); err != nil {
					return fmt.Errorf("failed to delete duplicates from %s: %w", args[0], err)
				}
			} else {
				fmt.Println("operation cancelled")
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&purge, "purge", "p", false, "permanently expunge duplicates instead of moving them to trash")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

// CreateFolder generates a command to recursively create the requested imap folder in account
func CreateFolder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mkdir <path>",
		Short: "recursively create imap folder",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("you must provide the folder path to create as the first positional argument")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			account := cmd.Context().Value("account").(imaputils.Account)
			if err := imaputils.EnsureFolder(imaputils.SheDialer, account, args[0]); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
