package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/henrygrant/go-cli-example/structs"
	"github.com/spf13/cobra"
)

func lookupPokemon(query string) structs.Pokemon {
	var poke structs.Pokemon
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", query)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if response.StatusCode != 200 {
		log.Fatal("no pokemon found for query: " + query)
	}
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal([]byte(responseData), &poke)
	return poke
}

func outputPokemon(pokemon []structs.Pokemon, jsonFormat bool) {
	if jsonFormat {
		json, err := json.MarshalIndent(pokemon, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(json))
	} else {
		for _, poke := range pokemon {
			fmt.Println(poke.HumanReadable())
		}
	}
}

var pokemonCmd = &cobra.Command{
	Use:   "pokemon",
	Short: "Gotta fetch them all!",
	Long:  `Fetches Pokemon from the PokeAPI. You can search by name, number, or range of numbers.`,
	Run: func(cmd *cobra.Command, args []string) {
		searchType := ""
		for _, flagName := range []string{"name", "number", "range"} {
			if cmd.Flags().Changed(flagName) {
				searchType = flagName
			}
		}

		jsonFormat, err := cmd.Flags().GetBool("json")
		if err != nil {
			log.Fatal(err)
		}

		switch searchType {
		case "name":
			poke := lookupPokemon(cmd.Flag("name").Value.String())
			outputPokemon([]structs.Pokemon{poke}, jsonFormat)
		case "number":
			poke := lookupPokemon(cmd.Flag("number").Value.String())
			outputPokemon([]structs.Pokemon{poke}, jsonFormat)
		case "range":
			matched, err := regexp.MatchString(`^\d+-\d+$`, cmd.Flag("range").Value.String())
			if err != nil {
				log.Fatal(err)
			}
			if !matched {
				log.Fatal(errors.New("invalid range format. must be <number>-<number>"))
			}
			bounds := strings.Split(cmd.Flag("range").Value.String(), "-")
			lowbound, err := strconv.ParseInt(bounds[0], 10, 16)
			if err != nil {
				log.Fatal(err)
			}
			highbound, err := strconv.ParseInt(bounds[1], 10, 16)
			if err != nil {
				log.Fatal(err)
			}
			if lowbound < 1 || highbound > 1025 || lowbound > highbound {
				log.Fatal(errors.New("lower bound must be less than greater bound and both must be between 1 and 1025"))
			}

			var wg sync.WaitGroup
			var pokeList []structs.Pokemon
			for i := lowbound; i <= highbound; i++ {
				wg.Add(1)
				go func(num string) {
					pokeList = append(pokeList, lookupPokemon(num))
					defer wg.Done()
				}(strconv.FormatInt(i, 10))
			}
			wg.Wait()
			sort.Slice(pokeList, func(i int, j int) bool {
				a := pokeList[i]
				b := pokeList[j]
				return a.ID < b.ID
			})
			outputPokemon(pokeList, jsonFormat)
		default:
			cmd.Usage()
		}

		// validate := func(input string) error {
		// 	_, err := strconv.ParseInt(input, 10, 16)
		// 	if err != nil {
		// 		return errors.New("Invalid number")
		// 	}
		// 	return nil
		// }

		// prompt := promptui.Prompt{
		// 	Label:    "Find Pokemon by name, number, or range of numbers (ex: 1-151)",
		// 	Validate: validate,
		// }

		// result, err := prompt.Run()

		// if err != nil {
		// 	fmt.Printf("Prompt failed %v\n", err)
		// 	return
		// }

		// fmt.Printf("You choose %q\n", result)

	},
}

func init() {
	rootCmd.AddCommand(pokemonCmd)
	pokemonCmd.PersistentFlags().String("name", "", "A name of a Pokemon (mutually exclusive with number and range)")
	pokemonCmd.PersistentFlags().String("number", "", "A number of a Pokemon (mutually exclusive with name and range)")
	pokemonCmd.PersistentFlags().String("range", "", "A range of numbers of Pokemon (ex. 1-151) (mutually exclusive with name and number)")
	pokemonCmd.MarkFlagsMutuallyExclusive("name", "number", "range")
}
