package wago

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/vasylcode/wago/internal/model"
	"github.com/vasylcode/wago/internal/storage"
	"github.com/vasylcode/wago/internal/util"
)

func init() {
	// Category command
	categoryCmd := &cobra.Command{
		Use:     "category",
		Aliases: []string{"cat"},
		Short:   "Manage wallet categories",
		Long:    `Add and delete wallet categories.`,
		Run:     listCategories,
	}

	// Add subcommand
	addCategoryCmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new category",
		Long:  `Add a new category with the specified name.`,
		Args:  cobra.ExactArgs(1),
		Run:   addCategory,
	}

	// Delete subcommand
	delCategoryCmd := &cobra.Command{
		Use:   "del [name]",
		Short: "Delete a category",
		Long:  `Delete a category. All wallets with this category will have their category removed.`,
		Args:  cobra.ExactArgs(1),
		Run:   deleteCategory,
	}

	// Add subcommands to category command
	categoryCmd.AddCommand(addCategoryCmd)
	categoryCmd.AddCommand(delCategoryCmd)

	// Add category command to root command
	rootCmd.AddCommand(categoryCmd)
}

func addCategory(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	category := &model.Category{
		Name:  name,
		Color: generateRandomColor(),
	}

	if err := s.AddCategory(category); err != nil {
		er(fmt.Sprintf("Failed to add category: %v", err))
		return
	}

	fmt.Printf("Category '%s' added successfully with color %s\n", name, category.Color)
}

func deleteCategory(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	name := args[0]
	if err := s.DeleteCategory(name); err != nil {
		er(fmt.Sprintf("Failed to delete category: %v", err))
		return
	}

	fmt.Printf("Category '%s' deleted successfully\n", name)
}

func listCategories(cmd *cobra.Command, args []string) {
	s, err := storage.New()
	if err != nil {
		er(fmt.Sprintf("Failed to initialize storage: %v", err))
		return
	}

	categories := s.ListCategories()
	if len(categories) == 0 {
		fmt.Println("No categories found")
		return
	}

	fmt.Println(color.New(color.Bold).Sprint("Categories:"))
	for _, category := range categories {
		// Get the color name from the category
		colorName := category.Color
		
		// Create a colored box to represent the category color
		colorBox := "■ "
		
		// Create a colored category name using the category's color
		categoryName := category.Name
		
		// Use the color name to create a terminal color if possible
		if colorName != "" {
			// Get a terminal color that matches the color name
			termColor := util.GetTerminalColor(colorName, color.FgHiWhite)
			
			// Apply the color to the box and create a bold category name
			coloredBox := termColor.Sprint(colorBox)
			boldName := color.New(color.Bold).Sprint(categoryName)
			
			// Print with the color box and the color name
			fmt.Printf("  %s%s (%s)\n", coloredBox, boldName, colorName)
		} else {
			// Default display if no color is set
			fmt.Printf("  %s (%s)\n", categoryName, "no color")
		}
	}
}

// generateRandomColor generates a random terminal color attribute
func generateRandomColor() string {
	rand.Seed(time.Now().UnixNano())
	
	// Define a set of vibrant terminal colors
	colors := []string{
		// Standard colors
		"red",
		"green",
		"yellow",
		"blue",
		"magenta",
		"cyan",
		"white",
		// Bright/high-intensity colors
		"brightred",
		"brightgreen",
		"brightyellow",
		"brightblue",
		"brightmagenta",
		"brightcyan",
		"brightwhite",
	}
	
	// Pick a random color from the list
	colorName := colors[rand.Intn(len(colors))]
	
	return colorName
}
