package wago

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
)

var (
	contactAddress string
	contactChain   string
	contactNote    string
)

func init() {
	// Contact command
	contactCmd := &cobra.Command{
		Use:     "contact",
		Aliases: []string{"c"},
		Short:   "Manage contacts",
		Long:    `Add, delete, and list contacts.`,
		Run:     listContacts,
	}

	// Add subcommand
	addContactCmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new contact",
		Long:  `Add a new contact with the specified name and properties.`,
		Args:  cobra.ExactArgs(1),
		Run:   addContact,
	}

	// Delete subcommand
	delContactCmd := &cobra.Command{
		Use:   "del [name]",
		Short: "Delete a contact",
		Long:  `Delete a contact.`,
		Args:  cobra.ExactArgs(1),
		Run:   deleteContact,
	}

	// Add flags to add command
	addContactCmd.Flags().StringVarP(&contactAddress, "address", "a", "", "Contact address")
	addContactCmd.Flags().StringVarP(&contactChain, "chain", "n", "", "Blockchain")
	addContactCmd.Flags().StringVarP(&contactNote, "note", "", "", "Note to describe contact")

	addContactCmd.MarkFlagRequired("address")
	addContactCmd.MarkFlagRequired("chain")

	// Add subcommands to contact command
	contactCmd.AddCommand(addContactCmd)
	contactCmd.AddCommand(delContactCmd)

	// Add contact command to root command
	rootCmd.AddCommand(contactCmd)
}

func addContact(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	contact := &model.Contact{
		Name:    name,
		Address: contactAddress,
		Chain:   contactChain,
		Note:    contactNote,
	}

	if err := s.AddContact(contact); err != nil {
		er(fmt.Sprintf("Failed to add contact: %v", err))
		return
	}

	fmt.Printf("Contact '%s' added successfully\n", name)
}

func deleteContact(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	if err := s.DeleteContact(name); err != nil {
		er(fmt.Sprintf("Failed to delete contact: %v", err))
		return
	}

	fmt.Printf("Contact '%s' deleted successfully\n", name)
}

func listContacts(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	contacts := s.ListContacts()
	if len(contacts) == 0 {
		fmt.Println("No contacts found")
		return
	}

	fmt.Println("Contacts:")
	for _, contact := range contacts {
		noteStr := ""
		if contact.Note != "" {
			noteStr = fmt.Sprintf(" (%s)", contact.Note)
		}
		
		fmt.Printf("  %s (%s) %s%s\n", 
			contact.Name, 
			contact.Address, 
			contact.Chain,
			noteStr)
	}
}
