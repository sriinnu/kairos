package main

import (
	"fmt"
	"os"

	"github.com/kairos/internal/visualization"
	"github.com/spf13/cobra"
)

var visualizer *visualization.Visualizer

var visualizeCmd = &cobra.Command{
	Use:   "visualize [week|month|html]",
	Short: "Generate visual reports",
	Long:  `Generate SVG or HTML visualizations of your work hours.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		visualizer = visualization.New()

		switch args[0] {
		case "week":
			return generateWeekSVG()
		case "month":
			return generateMonthSVG()
		case "html":
			output, _ := c.Flags().GetString("output")
			return generateHTMLReport(output)
		default:
			return fmt.Errorf("unknown visualization type: %s (use: week, month, or html)", args[0])
		}
	},
}

func generateWeekSVG() error {
	progress, err := trackerService.GetWeeklyProgress()
	if err != nil {
		return err
	}

	svg := visualizer.GenerateWeekSVG(progress)
	fmt.Println(svg)
	return nil
}

func generateMonthSVG() error {
	progress, err := trackerService.GetMonthlyProgress()
	if err != nil {
		return err
	}

	svg := visualizer.GenerateMonthSVG(progress)
	fmt.Println(svg)
	return nil
}

func generateHTMLReport(output string) error {
	dayProgress, err := trackerService.GetTodayProgress()
	if err != nil {
		return err
	}

	weekProgress, err := trackerService.GetWeeklyProgress()
	if err != nil {
		return err
	}

	html := visualizer.GenerateHTMLReport(dayProgress, weekProgress)

	if output != "" {
		return os.WriteFile(output, []byte(html), 0644)
	}

	fmt.Println(html)
	return nil
}

func init() {
	rootCmd.AddCommand(visualizeCmd)
	visualizeCmd.Flags().StringP("output", "o", "", "Output file path")
}
