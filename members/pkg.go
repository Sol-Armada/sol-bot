package members

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
)

type Member struct {
	Id          string     `json:"id"`
	Name        string     `json:"name"`
	Rank        ranks.Rank `json:"rank"`
	Notes       string     `json:"notes"`
	Avatar      string     `json:"avatar"`
	Updated     time.Time  `json:"updated"`
	ValidatedAt *time.Time `json:"validated_at"`
	Joined      time.Time  `json:"joined"`
	Suffix      string     `json:"suffix"`
	DmOptOut    bool       `json:"dm_opt_out"`
	DateLeft    *time.Time `json:"date_left"`

	RsiInfo *RsiInfo `json:"rsi_info"`

	MemberSince time.Time `json:"member_since"`

	IsBot       bool `json:"is_bot"`
	IsAlly      bool `json:"is_ally"`
	IsAffiliate bool `json:"is_affiliate"`

	DKP      []string `json:"dkp"`
	DKPSpent int      `json:"dkp_spent"`

	Merits   []*Merit   `json:"merits"`
	Demerits []*Demerit `json:"demerits"`

	BlueprintIds []string `json:"blueprintIds"`

	// onboarding info
	OnboardedAt *time.Time     `json:"onboarded_at"`
	Age         int            `json:"age"`
	Pronouns    string         `json:"pronouns"`
	Playtime    int            `json:"playtime"`
	Gameplay    []GameplayType `json:"gameplay"`
	Recruiter   *string        `json:"recruiter"`
	ChannelId   string         `json:"channel_id"`
	MessageId   string         `json:"message_id"`
	FoundBy     string         `json:"found_by"`
	TimeZone    string         `json:"time_zone"`
	Other       string         `json:"other"`

	LegacyAge       string `json:"legacy_age"`
	LegacyPlaytime  string `json:"legacy_playtime"`
	LegacyGameplay  string `json:"legacy_gameplay"`
	LegacyRecruiter string `json:"legacy_recruiter"`
	LegacyOther     string `json:"legacy_other"`
}

type RsiInfo struct {
	Handle        string   `json:"handle"`
	PrimaryOrg    string   `json:"primary_org"`
	PrimaryOrgSid string   `json:"primary_org_sid"`
	Affiliations  []string `json:"affiliations"`
}
type AttendedEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Merit struct {
	Giver  Member    `json:"giver"`
	Reason string    `json:"reason"`
	When   time.Time `json:"when"`
}

type Demerit struct {
	Giver  Member    `json:"giver"`
	Reason string    `json:"reason"`
	When   time.Time `json:"when"`
}

type MemberError error

var (
	MemberNotFound MemberError = errors.New("member not found")
)

var membersBackend memberBackend

func Setup() error {
	return setupMembersBackend()
}

func New(discordMember *discordgo.Member) *Member {
	m := &Member{
		Id:     discordMember.User.ID,
		Avatar: discordMember.Avatar,
		Joined: discordMember.JoinedAt.UTC(),
	}

	m.Name = m.GetTrueNick(discordMember)

	return m
}

func Get(id string) (*Member, error) {
	member, err := membersBackend.Get(id)
	if err != nil {
		return nil, err
	}
	rsiInfo, err := membersBackend.GetRsiInfo(member.Name)
	if err != nil {
		return nil, err
	}
	member.RsiInfo = rsiInfo
	return member, nil
}

func GetList(ids []string) ([]*Member, error) {
	members, err := membersBackend.GetList(ids)
	if err != nil {
		return nil, err
	}

	rsiInfoList, err := membersBackend.ListRsiInfoByHandles(extractNames(members))
	if err != nil {
		return nil, err
	}

	rsiInfoMap := make(map[string]*RsiInfo)
	for _, rsiInfo := range rsiInfoList {
		rsiInfoMap[rsiInfo.Handle] = rsiInfo
	}

	for _, member := range members {
		if rsiInfo, ok := rsiInfoMap[member.Name]; ok {
			member.RsiInfo = rsiInfo
		}
	}

	return members, nil
}

func GetRandom(max int, maxRank ranks.Rank) ([]Member, error) {
	return membersBackend.GetRandom(max, maxRank)
}

func List(ctx context.Context, page, limit int) ([]Member, error) {
	return membersBackend.List(ctx, page, limit)
}

func ListAll(ctx context.Context) ([]Member, error) {
	return membersBackend.ListAll(ctx)
}

func ListByBlueprint(blueprintId string) ([]Member, error) {
	return membersBackend.ListByBlueprint(blueprintId)
}

func ListPromotions() ([]dbgen.ListPromotionsRow, error) {
	return membersBackend.ListPromotions()
}

func (m *Member) GetTrueNick(discordMember *discordgo.Member) string {
	if discordMember == nil {
		return m.Name
	}

	trueNick := discordMember.User.Username
	if discordMember.Nick != "" {
		regRank := regexp.MustCompile(`\[(.*?)\] `)
		regAlly := regexp.MustCompile(`\{(.*?)\} `)
		suffix := regexp.MustCompile(` \((.*?)\)`)
		s := strings.TrimSpace(suffix.FindString(discordMember.Nick))
		s = strings.ReplaceAll(s, "(", "")
		s = strings.ReplaceAll(s, ")", "")
		m.Suffix = s
		trueNick = regRank.ReplaceAllString(discordMember.Nick, "")
		trueNick = regAlly.ReplaceAllString(trueNick, "")
		trueNick = suffix.ReplaceAllString(trueNick, "")
	}

	return trueNick
}

func (m *Member) Save() error {
	return membersBackend.Upsert(m)
}

func BulkSave(members []Member) error {
	if len(members) == 0 {
		return nil
	}
	now := time.Now().UTC()

	for i := range members {
		members[i].Updated = now
	}

	return membersBackend.BulkUpsert(members)
}

func GetStoredMemberIDs() ([]string, error) {
	return membersBackend.GetIDsOnly()
}

func ValidateDiscordMembers(discordMembers []*discordgo.Member) []*discordgo.Member {
	validMembers := make([]*discordgo.Member, 0, len(discordMembers))

	for _, member := range discordMembers {
		if member == nil || member.User == nil || member.User.Bot {
			continue
		}
		if !hasRankRole(member.Roles) {
			continue
		}
		validMembers = append(validMembers, member)
	}

	return validMembers
}

func hasRankRole(roles []string) bool {
	return slices.ContainsFunc(roles, func(role string) bool {
		switch role {
		case settings.GetString("DISCORD.ROLE_IDS.ALLY"),
			settings.GetString("DISCORD.ROLE_IDS.RECRUIT"),
			settings.GetString("DISCORD.ROLE_IDS.AFFILIATE"),
			settings.GetString("DISCORD.ROLE_IDS.MEMBER"),
			settings.GetString("DISCORD.ROLE_IDS.TECHNICIAN"),
			settings.GetString("DISCORD.ROLE_IDS.SPECIALIST"),
			settings.GetString("DISCORD.ROLE_IDS.LIEUTENANT"),
			settings.GetString("DISCORD.ROLE_IDS.COMMANDER"),
			settings.GetString("DISCORD.ROLE_IDS.ADMIRAL"):
			return true
		default:
			return false
		}
	})
}

func (m *Member) IsAdmin() bool {
	logger := slog.Default().With("id", m.Id)
	logger.Debug("checking if admin")
	if m.Rank <= ranks.Lieutenant {
		return true
	}
	logger.Debug("is NOT admin")
	return false
}

func (m *Member) Delete(reason string) error {
	log.WithField("member", m).Debug("deleting member")

	return membersBackend.Delete(m.Id, reason)
}

func (m *Member) ToMap() map[string]any {
	r := map[string]any{}
	b, _ := json.Marshal(m)
	_ = json.Unmarshal(b, &r)
	return r
}

func (m *Member) GiveMerit(reason string, who *Member) error {
	m.Merits = append(m.Merits, &Merit{
		Giver:  *who,
		Reason: reason,
		When:   time.Now().UTC(),
	})

	return m.Save()
}

func (m *Member) GiveDemerit(reason string, who *Member) error {
	m.Demerits = append(m.Demerits, &Demerit{
		Giver:  *who,
		Reason: reason,
		When:   time.Now().UTC(),
	})

	return m.Save()
}

func (m *Member) GetOnboardingMessage() *discordgo.Message {
	onboarded := "No"
	if m.OnboardedAt != nil {
		onboarded = m.OnboardedAt.Format("2006-01-02 15:04:05 -0700 MST")
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Onboarded",
			Value: onboarded,
		},
	}

	if m.OnboardedAt != nil {
		fields = append(fields, []*discordgo.MessageEmbedField{
			{
				Name:  "Age",
				Value: "Not Set",
			},
			{
				Name:  "RSI Profile",
				Value: "None",
			},
			{
				Name:  "Primary set Org",
				Value: "None",
			},
			{
				Name:  "Affiliated Orgs",
				Value: "None",
			},
			{
				Name:  "How they found us",
				Value: "A way",
			},
			{
				Name:  "Who Recruited",
				Value: "No one",
			},
			{
				Name:  "Time Playing Star Citizen",
				Value: "Many Years",
			},
			{
				Name:  "Interested Gameplay",
				Value: "Salvage, Bounty Hunting, Mining",
			},
		}...)
	}

	description := ""
	if m.DateLeft != nil {
		description = "Left " + m.DateLeft.Format("2006-01-02 15:04:05 -0700 MST")
	}

	message := &discordgo.Message{
		Content: "<@" + m.Id + ">",
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Info",
				Description: description,
				Fields:      fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Joined " + m.Joined.Format("2006-01-02 15:04:05 -0700 MST"),
				},
			},
		},
	}

	return message
}

func (m *Member) IsRanked() bool {
	return m.Rank <= ranks.Recruit
}

func (m *Member) IsOfficer() bool {
	return m.Rank <= ranks.Lieutenant
}

func (m *Member) UpdateRank(discordRoles []string) {
	m.Rank = ranks.Guest
	rankIds := ranks.GetRoleIDs()
	for rankName, roleID := range rankIds {
		rank := ranks.GetRankByName(rankName)
		if rank != ranks.None && slices.Contains(discordRoles, roleID) && rank < m.Rank {
			m.Rank = rank
		}
	}
}

func (m *Member) UpdateRsiInfo() error {
	info, err := rsi.GetRSIInfo(m.Name)
	if err != nil {
		return err
	}

	affiliations := make([]string, len(info.Affiliation))
	for i, aff := range info.Affiliation {
		affiliations[i] = aff.Name
	}

	updatedRsiInfo := &RsiInfo{
		Handle:        m.Name,
		PrimaryOrg:    info.Organization.Name,
		PrimaryOrgSid: info.Organization.SID,
		Affiliations:  affiliations,
	}

	if rsiInfoEqual(m.RsiInfo, updatedRsiInfo) {
		m.RsiInfo = updatedRsiInfo
		return nil
	}

	m.RsiInfo = updatedRsiInfo
	return m.RsiInfo.Save()
}

func rsiInfoEqual(a, b *RsiInfo) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Handle == b.Handle &&
		a.PrimaryOrg == b.PrimaryOrg &&
		a.PrimaryOrgSid == b.PrimaryOrgSid &&
		slices.Equal(a.Affiliations, b.Affiliations)
}

func (m *Member) OnRsi() bool {
	return m.RsiInfo != nil
}

func (m *Member) BadAffiliation() bool {
	if m.RsiInfo == nil {
		return false
	}
	for _, aff := range m.RsiInfo.Affiliations {
		if slices.Contains(settings.GetStringSlice("ENEMIES"), aff) {
			return true
		}
	}
	return false
}

func (r *RsiInfo) Save() error {
	return membersBackend.UpsertRsiInfo(r)
}

func (m *Member) OptOutOfDMs() error {
	m.DmOptOut = true
	return m.Save()
}

func (m *Member) UpdateFromDiscordMember(discordMember *discordgo.Member) error {
	changed := m.ApplyDiscordMember(discordMember)
	if changed {
		return m.Save()
	}
	return nil
}

// ApplyDiscordMember updates member fields from Discord data and returns whether
// any member fields changed.
func (m *Member) ApplyDiscordMember(discordMember *discordgo.Member) bool {
	original := *m

	truenick := m.GetTrueNick(discordMember)
	m.Name = strings.ReplaceAll(truenick, ".", "")

	if m.Joined.IsZero() {
		m.Joined = discordMember.JoinedAt.UTC()
	}

	m.Avatar = discordMember.User.Avatar

	m.UpdateRank(discordMember.Roles)

	return !reflect.DeepEqual(original, *m)
}

func extractNames(members []*Member) []string {
	names := make([]string, 0, len(members))
	for _, member := range members {
		if member != nil {
			names = append(names, member.Name)
		}
	}
	return names
}

func GetPromotionsEmbed() (*discordgo.MessageEmbed, error) {
	// get promotions
	promotions, err := ListPromotions()
	if err != nil {
		return nil, err
	}

	if len(promotions) == 0 {
		return nil, customerrors.NoPromotions
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Members to Rank Up",
			Value:  "",
			Inline: true,
		},
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Promotions Report",
		Fields: fields,
	}

	ind := 0
	for _, promotion := range promotions {
		if promotion.NextRank == 0 {
			continue
		}

		if ind%10 == 0 && ind != 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Members to Rank Up (continued)",
				Value:  "",
				Inline: true,
			})
		}

		field := fields[len(fields)-1]
		field.Value += fmt.Sprintf("<@%s> to %s (%d Events)", promotion.ID, ranks.Rank(promotion.NextRank).String(), promotion.AttendanceCount)

		// if not the 10th member, add a newline
		if ind%10 != 9 {
			field.Value += "\n"
		}

		ind++
	}

	return embed, nil
}
