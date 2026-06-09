package cli

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/spf13/cobra"
	"github.com/wryfi/shemail/imaputils"
	"github.com/wryfi/shemail/util"
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

			if table, err := util.TabulateMessages(messages); err == nil {
				table.Render()
			} else {
				return fmt.Errorf("error tabulating messages: %w", err)
			}

			// confirm honors --yes for non-interactive use (e.g. cron).
			confirm := func(prompt string) bool {
				return assumeYes || util.GetConfirmation(prompt)
			}

			if moveTo != "" {
				if confirm(fmt.Sprintf("really move %d messages to %s?", len(messages), moveTo)) {
					err := imaputils.MoveMessages(imaputils.SheDialer, account, messages, args[0], moveTo, 100)
					if err != nil {
						return fmt.Errorf("failed to move messages to %s: %w", moveTo, err)
					}
				} else {
					fmt.Println("operation cancelled")
				}
			}

			if copyTo != "" {
				if confirm(fmt.Sprintf("really copy %d messages to %s?", len(messages), copyTo)) {
					if err := imaputils.CopyMessages(imaputils.SheDialer, account, messages, args[0], copyTo); err != nil {
						return fmt.Errorf("failed to copy messages to %s: %w", copyTo, err)
					}
				} else {
					fmt.Println("operation cancelled")
				}
			}

			if markRead || markUnread {
				state := "read"
				if markUnread {
					state = "unread"
				}
				if confirm(fmt.Sprintf("really mark %d messages as %s?", len(messages), state)) {
					if err := imaputils.MarkMessages(imaputils.SheDialer, account, messages, args[0], markRead); err != nil {
						return fmt.Errorf("failed to mark messages as %s: %w", state, err)
					}
				} else {
					fmt.Println("operation cancelled")
				}
			}

			if deleteFrom {
				// --purge overrides the account setting for this run, expunging
				// in place instead of moving to a trash folder.
				account.Purge = account.Purge || purge
				action := "delete"
				if account.Purge {
					action = "permanently delete"
				}
				if confirm(fmt.Sprintf("really %s %d messages from %s?", action, len(messages), args[0])) {
					err := imaputils.DeleteMessages(imaputils.SheDialer, account, messages, args[0])
					if err != nil {
						return fmt.Errorf("failed to delete messages from %s: %w", args[0], err)
					}
				} else {
					fmt.Println("operation cancelled")
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
