package attendance

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/admin/members"
)

type AttendanceIssue struct {
	Member *members.Member
	Reason string
}

type Attendance struct {
	Id      string
	Name    string
	Members []*members.Member
	Issues  []*AttendanceIssue
}

func (a *Attendance) NewFromThreadMessages(threadMessages []*discordgo.Message) {
	mainMessage := threadMessages[len(threadMessages)-1].ReferencedMessage
	attendanceMessage := threadMessages[len(threadMessages)-2]

	// get the ID between ( )
	reg := regexp.MustCompile(`(.*?)\((.*?)\)`)
	a.Id = reg.FindStringSubmatch(mainMessage.Content)[1]

	// get the name before ( )
	a.Name = reg.FindStringSubmatch(mainMessage.Content)[0]

	currentUsersSplit := strings.Split(attendanceMessage.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(attendanceMessage.Embeds[0].Fields[0].Value, "\n")...)
	for _, cu := range currentUsersSplit[1:] {
		if cu == "No members" || cu == "" {
			continue
		}
		uid := strings.ReplaceAll(cu, "<@", "")
		uid = strings.ReplaceAll(uid, ">", "")
		uid = strings.Split(uid, ":")[0]

		member, err := members.Get(uid)
		if err != nil {
			log.WithError(err).Error("getting user for existing attendance")
			return
		}

		if len(Issues(member)) > 0 {
			a.Issues = append(a.Issues, &AttendanceIssue{
				Member: member,
				Reason: strings.Join(Issues(member), ", "),
			})
			continue
		}

		a.Members = append(a.Members, member)
	}
}

func (a *Attendance) AddMember(member *members.Member) {
	a.Members = append(a.Members, member)
	a.removeDuplicates()
}

func (a *Attendance) GenerateList() string {
	// remove duplicates
	list := make(map[string]*members.Member)
	for _, u := range a.Members {
		list[u.Id] = u
	}

	a.Members = []*members.Member{}
	for _, u := range list {
		a.Members = append(a.Members, u)
	}

	slices.SortFunc(a.Members, func(a, b *members.Member) int {
		if a.Rank > b.Rank {
			return 1
		}
		if a.Rank < b.Rank {
			return -1
		}
		if a.Name < b.Name {
			return 1
		}
		if a.Name > b.Name {
			return -1
		}

		return 0
	})

	m := ""
	for i, u := range a.Members {
		m += fmt.Sprintf("<@%s>", u.Id)
		if i < len(a.Members)-1 {
			m += "\n"
		}
	}

	if m == "" {
		m = "No members"
	}

	return "Attendance List:\n" + m
}

func (a *Attendance) removeDuplicates() {
	list := []*members.Member{}

	for _, u := range a.Members {
		found := false
		for _, v := range list {
			if u.Id == v.Id {
				found = true
				break
			}
		}
		if found {
			continue
		}
		list = append(list, u)
	}

	a.Members = list
}

func (a *Attendance) GetIssuesEmbed() *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "Users with Issues",
		Description: "List of members with attendance credit issues",
		Fields:      []*discordgo.MessageEmbedField{},
	}

	fieldValue := ""
	for _, issue := range a.Issues {
		fieldValue += fmt.Sprintf("<@%s>: %s\n", issue.Member.Id, issue.Reason)
	}
	field := &discordgo.MessageEmbedField{
		Name:  "Member - Issues",
		Value: fieldValue,
	}
	embed.Fields = append(embed.Fields, field)

	return embed
}
