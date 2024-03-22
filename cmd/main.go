package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"tpf/pkg/filter"
)

var (
	filterConfig map[string]map[string]any
	cfgFile      string
	eot          bool
	diff         bool
	rootCmd      = &cobra.Command{
		Use:     "tpf",
		Example: "terraform show terraform.tfplan | tpf -f filter.yaml",
		Short:   "tpf is a tool to filter terraform plan output",
		Long:    `tpf is a small utility to filter out specific data from terraform plan output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := mustPipe()
			if err != nil {
				return err
			}
			planLines, err := getPlanLines()
			if err != nil {
				return err
			}
			err = filter.Execute(planLines, filterConfig, eot, diff)
			if err != nil {
				return err
			}
			return nil
		},
	}
)

func main() {
	rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "file", "f", "", "the location of config file that contains filter rules")
	rootCmd.PersistentFlags().BoolVarP(&eot, "eot", "e", false, "hide EOT blocks globally")
	rootCmd.PersistentFlags().BoolVarP(&diff, "diff", "d", false, "convert to diff globally")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		path, err := os.Getwd()
		cobra.CheckErr(err)

		viper.AddConfigPath(path)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".tpf.yaml")
	}

	viper.AutomaticEnv()
	filterConfig := make(map[string]any)
	if err := viper.ReadInConfig(); err == nil {
		for key, _ := range viper.AllSettings() {
			value := viper.GetStringMap(key)
			for k, v := range value {
				filterConfig[k] = v
			}
		}
	} else {
		log.Fatalln(err)
	}
}

// Ensure the program receives the input via a pipe
func mustPipe() error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return fmt.Errorf("error binding to stdin: %v", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return errors.New("please use a pipe to pass the input")
	}
	return nil
}

func getPlanLines() (string, error) {
	bytes, err := io.ReadAll(os.Stdin)
	return string(bytes), err
}
