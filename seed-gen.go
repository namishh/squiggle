//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"
)

var firstNames = []string{
	"Alex", "Sam", "Jordan", "Taylor", "Casey", "Morgan", "Riley", "Jamie",
	"Avery", "Quinn", "Drew", "Skyler", "Reese", "Rowan", "Finley", "Emerson",
	"Kai", "Sage", "Blake", "Charlie",
}

var lastNames = []string{
	"Carter", "Bennett", "Hayes", "Morrison", "Sullivan", "Bishop", "Pierce",
	"Whitfield", "Sinclair", "Donovan", "Ellis", "Fletcher", "Griffin", "Harmon",
	"Kessler", "Lawson", "Mercer", "Norwood", "Prescott", "Sawyer",
}

var positiveMessages = []string{
	"This site is amazing, love the design!",
	"Really clean layout, great work.",
	"Been following for a while, keep it up!",
	"Wow this is such a nice personal site.",
	"Love the writing style here.",
	"This mf website is so good, well done.",
	"Super impressed with the projects section.",
	"Genuinely one of the best personal sites I've seen.",
	"The color scheme is *chef's kiss*.",
	"Bookmarking this immediately, great content.",
	"You clearly put a lot of care into this.",
	"This inspired me to redo my own site.",
	"Such a refreshing take on a portfolio site.",
	"The blog posts are genuinely insightful.",
	"Nice touch with the little animations.",
	"Loading speed is incredible, well optimized.",
	"This is exactly what a personal site should look like.",
	"Your writing has such a distinct voice, love it.",
	"Followed you after reading just one post.",
	"Clean code, clean design, great job overall.",
}

var neutralMessages = []string{
	"The font is a bit hard to read on mobile.",
	"Would be nice to have a dark mode.",
	"Not bad, could use some polish.",
	"Interesting content, a bit slow to load though.",
	"Decent overall, nothing groundbreaking.",
	"Some sections feel a little empty.",
	"Navigation could be more intuitive.",
	"It's fine, does what it needs to.",
	"A few broken links here and there.",
	"Could use more contrast in the text.",
	"The about page felt a bit short.",
	"Mobile version needs some work.",
	"Simple site, gets the job done.",
	"Wish there were more project details.",
	"Layout is fine but a bit generic.",
	"Nothing wrong with it, just not memorable.",
	"Could benefit from a proper footer.",
	"The spacing feels a little inconsistent.",
	"It works, though I've seen better.",
	"Average site, does the basics well.",
}

var negativeMessages = []string{
	"This is garbage, whoever made this is an idiot.",
	"Terrible design, waste of time.",
	"Absolutely hate this layout.",
	"Worst site I've seen this week.",
	"You clearly have no idea what you're doing.",
	"This looks like it was made in five minutes.",
	"Total eyesore, fix your color choices.",
	"Why does this even exist.",
	"Unusable on mobile, complete failure.",
	"This is an insult to web design.",
	"Whoever approved this design should be fired.",
	"Painfully bad, do better.",
	"This site actively made my day worse.",
	"Embarrassing effort honestly.",
	"I want those five minutes of my life back.",
	"This is the worst thing I've seen online.",
	"Completely broken and badly designed.",
	"You should be ashamed of this.",
	"No effort was put into this at all.",
	"Absolute trainwreck of a website.",
}

var sarcasticMessages = []string{
	"Oh wow, another portfolio site, so original.",
	"Love what you've done with... whatever this is.",
	"Truly groundbreaking use of a template.",
	"Wow, a contact form, never seen that before.",
	"Really outdoing yourself with this one, huh.",
	"Nice, another site that takes 10 seconds to load.",
	"Very impressive, if this were 2010.",
	"Bold choice going with default fonts.",
	"Ah yes, the classic 'under construction' vibe.",
	"Incredible, you found the world's most basic layout.",
	"Wow, comic sans, brave choice.",
	"Really pushing the boundaries of web design here.",
	"So cutting edge, very Web 2.0 of you.",
	"Love the 'I gave up halfway' aesthetic.",
	"Truly the pinnacle of minimalism, or laziness.",
	"Great job copying every other portfolio site.",
	"Very unique, said no one ever.",
	"Wow, a hero section, how avant-garde.",
	"This must have taken you all of ten minutes.",
	"Riveting stuff, truly a page-turner.",
}

var sites = []string{
	"https://example.com", "https://mysite.dev", "https://blog.example.org", "",
}

func randomFrom(list []string) string {
	return list[rand.Intn(len(list))]
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip + "seed-salt"))
	return hex.EncodeToString(sum[:])
}

func randomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
}

func escape(s string) string {
	out := ""
	for _, c := range s {
		if c == '\'' {
			out += "''"
		} else {
			out += string(c)
		}
	}
	return out
}

func main() {

	entryCount := 2000

	f, err := os.Create("./sql/seed.sql")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "-- GENERATED FROM `seed-gen.go`. DO NOT EDIT ")
	fmt.Fprintln(f, "BEGIN;")

	// entries
	ipPool := make([]string, 200)
	for i := range ipPool {
		ipPool[i] = randomIP()
	}

	spamIPCounts := map[string]int{}

	for i := 0; i < entryCount; i++ {
		name := randomFrom(firstNames) + " " + randomFrom(lastNames)
		email := fmt.Sprintf("%s%d@example.com", name, rand.Intn(9999))
		site := randomFrom(sites)
		ip := ipPool[rand.Intn(len(ipPool))]
		ipHash := hashIP(ip)
		userAgent := "Mozilla/5.0 (SyntheticBot/1.0)"

		roll := rand.Float64()
		var message string
		var score float64
		var status string

		switch {
		case roll < 0.6:
			message = randomFrom(positiveMessages)
			score = 15 + rand.Float64()*5 // 15-20
			status = "visible"
		case roll < 0.85:
			message = randomFrom(neutralMessages)
			score = 9 + rand.Float64()*6 // 9-15
			status = "hidden"
		default:
			message = randomFrom(negativeMessages)
			score = rand.Float64() * 4 // 0-4
			status = "spam"
			spamIPCounts[ipHash]++
		}

		daysAgo := rand.Intn(180)
		createdAt := time.Now().AddDate(0, 0, -daysAgo).Format("2006-01-02 15:04:05")

		fmt.Fprintf(f, `INSERT INTO entries (name, email, site, message, status, ip_hash, user_agent, sentiment_score, created_at) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', %.2f, '%s');`+"\n",
			escape(name), escape(email), escape(site), escape(message), status, ipHash, userAgent, score, createdAt)
	}

	// defaulters — derived from actual spam entries above, so ip_hash values correlate across tables
	for ipHash, count := range spamIPCounts {
		if count < 2 {
			continue // only track repeat offenders, not one-off spam
		}
		email := fmt.Sprintf("defaulter-%s@example.com", ipHash[:8])
		banned := count >= 5
		lastOffense := time.Now().AddDate(0, 0, -rand.Intn(30)).Format("2006-01-02 15:04:05")

		fmt.Fprintf(f, `INSERT INTO defaulters (ip_hash, email, low_sentiment_count, banned, last_offense_at) VALUES ('%s', '%s', %d, %t, '%s') ON CONFLICT (ip_hash) DO NOTHING;`+"\n",
			ipHash, email, count, banned, lastOffense)
	}

	fmt.Fprintln(f, "COMMIT;")

	fmt.Println("Generated seed.sql with", entryCount, "entries and", len(spamIPCounts), "candidate defaulter IPs")
}
