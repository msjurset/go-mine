package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/msjurset/golars"

	"github.com/msjurset/go-mine/app"
)

var version = "dev"

func main() {
	generate := flag.Bool("generate", false, "Generate sample data instead of loading a file")
	rows := flag.Int("rows", 0, "Number of rows to generate (with -generate)")
	info := flag.Bool("info", false, "Print data summary to stdout and exit (non-interactive)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	completion := flag.String("completion", "", "Print shell completion script (zsh, bash)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "go-mine %s - Interactive data explorer powered by golars\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  go-mine <file.csv|file.parquet|file.json|file.tsv>\n")
		fmt.Fprintf(os.Stderr, "  go-mine -generate [-rows N]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nInteractive keys:\n")
		fmt.Fprintf(os.Stderr, "  1-5/tab    Switch views: Table, Stats, Filter, SQL, Columns\n")
		fmt.Fprintf(os.Stderr, "  j/k ↑↓     Navigate rows\n")
		fmt.Fprintf(os.Stderr, "  h/l ←→     Scroll columns\n")
		fmt.Fprintf(os.Stderr, "  s          Sort by current column (cycles: asc → desc → none)\n")
		fmt.Fprintf(os.Stderr, "  pgup/pgdn  Page through data\n")
		fmt.Fprintf(os.Stderr, "  g/G        Jump to top/bottom\n")
		fmt.Fprintf(os.Stderr, "  q          Quit\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("go-mine %s\n", version)
		return
	}

	if *completion != "" {
		switch *completion {
		case "zsh":
			fmt.Print(zshCompletion)
		case "bash":
			fmt.Print(bashCompletion)
		default:
			fmt.Fprintf(os.Stderr, "unsupported shell: %s (supported: zsh, bash)\n", *completion)
			os.Exit(1)
		}
		return
	}

	var df *golars.DataFrame
	var err error
	fileName := "generated"

	if *generate {
		n := 10000
		if *rows > 0 {
			n = *rows
		}
		df = generateSampleData(n)
	} else {
		args := flag.Args()
		if len(args) < 1 {
			flag.Usage()
			os.Exit(1)
		}

		path := args[0]
		fileName = filepath.Base(path)
		ext := strings.ToLower(filepath.Ext(path))

		switch ext {
		case ".csv", ".tsv":
			opts := []golars.ReadCSVOption{}
			if ext == ".tsv" {
				opts = append(opts, golars.WithSeparator('\t'))
			}
			df, err = golars.ReadCSV(path, opts...)
		case ".parquet":
			df, err = golars.ReadParquet(path)
		case ".json":
			df, err = golars.ReadJSON(path)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported file type: %s\nSupported: .csv, .tsv, .parquet, .json\n", ext)
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	if *info {
		printInfo(df, fileName)
		return
	}

	model := app.NewModel(df, fileName)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateSampleData(n int) *golars.DataFrame {
	firstNames := []string{
		"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace",
		"Hank", "Ivy", "Jack", "Karen", "Leo", "Mia", "Noah", "Olivia",
		"Peter", "Quinn", "Rachel", "Sam", "Tina", "Uma", "Victor", "Wendy",
		"Xander", "Yara", "Zoe", "Aaron", "Beth", "Chris", "Dana",
		"Elena", "Felix", "Gina", "Hugo", "Iris", "Jake", "Lena",
		"Marco", "Nina", "Oscar", "Priya", "Ravi", "Sofia", "Theo",
	}
	lastNames := []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
		"Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez",
		"Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
		"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark",
		"Ramirez", "Lewis", "Robinson", "Walker", "Young", "Allen", "King",
		"Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores", "Green",
		"Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell",
		"Carter", "Roberts", "Patel", "Chen", "Kim", "Singh", "Muller",
	}
	cities := []string{
		"New York", "San Francisco", "Chicago", "Seattle", "Austin",
		"Boston", "Denver", "Portland", "Miami", "Atlanta",
		"Los Angeles", "Houston", "Phoenix", "Philadelphia", "San Antonio",
		"San Diego", "Dallas", "Nashville", "Minneapolis", "Detroit",
		"London", "Berlin", "Tokyo", "Toronto", "Sydney",
	}
	countries := []string{
		"US", "US", "US", "US", "US", "US", "US", "US", "US", "US",
		"US", "US", "US", "US", "US", "US", "US", "US", "US", "US",
		"UK", "DE", "JP", "CA", "AU",
	}
	departments := []string{
		"Engineering", "Marketing", "Design", "Management", "Sales",
		"Support", "Finance", "Legal", "HR", "Research",
	}
	levels := []string{"Junior", "Mid", "Senior", "Staff", "Principal", "Director", "VP"}
	products := []string{
		"Platform", "Mobile App", "Analytics", "Infrastructure", "API",
		"Dashboard", "CLI Tool", "SDK", "Data Pipeline", "Auth Service",
	}
	statuses := []string{"Active", "On Leave", "Probation", "Contractor", "Intern"}
	educations := []string{"High School", "Associate", "Bachelor", "Master", "PhD", ""}

	ids := make([]int64, n)
	fullNames := make([]string, n)
	emails := make([]string, n)
	ages := make([]int64, n)
	cityCol := make([]string, n)
	countryCol := make([]string, n)
	salaries := make([]float64, n)
	bonuses := make([]float64, n)
	bonusValid := make([]bool, n)
	depts := make([]string, n)
	levelCol := make([]string, n)
	productCol := make([]string, n)
	yearsExp := make([]int64, n)
	perfScores := make([]float64, n)
	perfValid := make([]bool, n)
	satisfaction := make([]int64, n)
	projects := make([]int64, n)
	remote := make([]bool, n)
	statusCol := make([]string, n)
	educationCol := make([]string, n)
	eduValid := make([]bool, n)
	teamSize := make([]int64, n)
	teamValid := make([]bool, n)
	overtime := make([]float64, n)

	for i := 0; i < n; i++ {
		ids[i] = int64(100000 + i)

		first := firstNames[rand.IntN(len(firstNames))]
		last := lastNames[rand.IntN(len(lastNames))]
		fullNames[i] = first + " " + last
		emails[i] = fmt.Sprintf("%s.%s@example.com",
			strings.ToLower(first), strings.ToLower(last))

		ages[i] = int64(20 + rand.IntN(45))

		cityIdx := rand.IntN(len(cities))
		cityCol[i] = cities[cityIdx]
		countryCol[i] = countries[cityIdx]

		dept := departments[rand.IntN(len(departments))]
		depts[i] = dept

		lvlIdx := rand.IntN(len(levels))
		levelCol[i] = levels[lvlIdx]

		// Salary correlates with level and department
		baseSalary := 45000.0 + float64(lvlIdx)*18000.0
		if dept == "Engineering" || dept == "Management" || dept == "Research" {
			baseSalary *= 1.15
		}
		salaries[i] = baseSalary + rand.Float64()*25000.0 - 5000.0

		// Bonus: 15% chance of null
		if rand.IntN(100) < 85 {
			bonuses[i] = salaries[i] * (0.02 + rand.Float64()*0.18)
			bonusValid[i] = true
		}

		productCol[i] = products[rand.IntN(len(products))]

		yearsExp[i] = int64(lvlIdx) + int64(rand.IntN(5))
		if yearsExp[i] > ages[i]-20 {
			yearsExp[i] = ages[i] - 20
		}
		if yearsExp[i] < 0 {
			yearsExp[i] = 0
		}

		// Performance score: 10% null
		if rand.IntN(100) < 90 {
			perfScores[i] = 1.0 + rand.Float64()*4.0 // 1.0 - 5.0
			perfValid[i] = true
		}

		satisfaction[i] = int64(1 + rand.IntN(10)) // 1-10
		projects[i] = int64(rand.IntN(12))
		remote[i] = rand.IntN(100) < 40
		statusCol[i] = statuses[rand.IntN(len(statuses))]

		// Education: 8% null
		if rand.IntN(100) < 92 {
			edu := educations[rand.IntN(len(educations)-1)] // skip empty
			educationCol[i] = edu
			eduValid[i] = true
		}

		// Team size: only for Senior+ (null for others)
		if lvlIdx >= 2 {
			teamSize[i] = int64(2 + rand.IntN(20))
			teamValid[i] = true
		}

		overtime[i] = float64(rand.IntN(200)) / 10.0 // 0.0 - 20.0 hrs/week
	}

	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", ids),
		golars.NewStringSeries("name", fullNames),
		golars.NewStringSeries("email", emails),
		golars.NewInt64Series("age", ages),
		golars.NewStringSeries("city", cityCol),
		golars.NewStringSeries("country", countryCol),
		golars.NewStringSeries("department", depts),
		golars.NewStringSeries("level", levelCol),
		golars.NewFloat64Series("salary", salaries),
		golars.NewFloat64SeriesWithValidity("bonus", bonuses, bonusValid),
		golars.NewStringSeries("product", productCol),
		golars.NewInt64Series("years_exp", yearsExp),
		golars.NewFloat64SeriesWithValidity("perf_score", perfScores, perfValid),
		golars.NewInt64Series("satisfaction", satisfaction),
		golars.NewInt64Series("projects", projects),
		golars.NewBooleanSeries("remote", remote),
		golars.NewStringSeries("status", statusCol),
		golars.NewStringSeriesWithValidity("education", educationCol, eduValid),
		golars.NewInt64SeriesWithValidity("team_size", teamSize, teamValid),
		golars.NewFloat64Series("overtime_hrs", overtime),
	)
	return df
}

func printInfo(df *golars.DataFrame, fileName string) {
	h, w := df.Shape()
	fmt.Printf("File: %s\n", fileName)
	fmt.Printf("Shape: %d rows × %d columns\n\n", h, w)

	fmt.Println("Schema:")
	schema := df.Schema()
	for i := 0; i < schema.Len(); i++ {
		f := schema.Field(i)
		fmt.Printf("  %-20s %s\n", f.Name, f.Dtype)
	}

	fmt.Println("\nHead (5 rows):")
	fmt.Println(df.Head(5))

	fmt.Println("\nDescribe:")
	desc := df.Describe()
	if desc != nil {
		fmt.Println(desc)
	}
}
