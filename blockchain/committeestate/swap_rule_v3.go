package committeestate

import (
	"github.com/incognitochain/incognito-chain/blockchain/signaturecounter"
	"github.com/incognitochain/incognito-chain/instruction"
)

//swapRuleV3 ...
type swapRuleV3 struct {
}

func NewSwapRuleV3() *swapRuleV3 {
	return &swapRuleV3{}
}

//GenInstructions generate instructions for swap rule v3
func (s *swapRuleV3) GenInstructions(
	shardID byte,
	committees, substitutes []string,
	minCommitteeSize, maxCommitteeSize, typeIns, numberOfFixedValidators int,
	penalty map[string]signaturecounter.Penalty,
) (*instruction.SwapShardInstruction, []string, []string, []string, []string) {

	//get slashed nodes
	newCommittees, slashingCommittees := s.slashingSwapOut(committees, penalty, minCommitteeSize, numberOfFixedValidators, MAX_SLASH_PERCENT)
	lenSlashedCommittees := len(slashingCommittees)
	//get normal swap out nodes
	newCommittees, normalSwapOutCommittees := s.normalSwapOut(
		newCommittees, substitutes, len(committees), lenSlashedCommittees, MAX_SWAP_OUT_PERCENT,
		numberOfFixedValidators, minCommitteeSize)
	swappedOutCommittees := append(slashingCommittees, normalSwapOutCommittees...)

	newCommittees, newSubstitutes, swapInCommittees :=
		s.swapInAfterSwapOut(newCommittees, substitutes, MAX_SWAP_IN_PERCENT, maxCommitteeSize,
			numberOfFixedValidators)

	if len(swapInCommittees) == 0 && len(swappedOutCommittees) == 0 {
		return instruction.NewSwapShardInstruction(), newCommittees, newSubstitutes, slashingCommittees, normalSwapOutCommittees
	}

	swapShardInstruction := instruction.NewSwapShardInstructionWithValue(
		swapInCommittees,
		swappedOutCommittees,
		int(shardID),
		typeIns,
	)

	return swapShardInstruction, newCommittees, newSubstitutes, slashingCommittees, normalSwapOutCommittees
}

func (s *swapRuleV3) AssignOffset(lenShardSubstitute, lenCommittees, numberOfFixedValidators, minCommitteeSize int) int {
	assignOffset := lenCommittees / MAX_ASSIGN_PERCENT
	if assignOffset == 0 && lenCommittees < MAX_ASSIGN_PERCENT {
		return 1
	}
	if lenCommittees-numberOfFixedValidators < assignOffset {
		assignOffset = 1
	}
	return assignOffset
}

func (s *swapRuleV3) swapInAfterSwapOut(
	committees, substitutes []string,
	maxSwapInPercent, numberOfFixedValidators,
	maxCommitteeSize int,
) (
	[]string, []string, []string,
) {
	resCommittees := committees
	resSubstitutes := substitutes
	resSwapInCommittees := []string{}
	swapInOffset := s.getSwapInOffset(len(committees), len(substitutes), maxSwapInPercent, maxCommitteeSize)

	resSwapInCommittees = append(resSwapInCommittees, substitutes[:swapInOffset]...)
	resSubstitutes = resSubstitutes[swapInOffset:]
	resCommittees = append(resCommittees, resSwapInCommittees...)

	return resCommittees, resSubstitutes, resSwapInCommittees
}

func (s *swapRuleV3) getSwapInOffset(
	lenCommitteesAfterSwapOut, lenSubstitutes int,
	maxSwapInPercent, maxCommitteeSize int,
) int {
	offset := lenCommitteesAfterSwapOut / maxSwapInPercent
	if lenSubstitutes < offset {
		return lenSubstitutes
	}
	if lenCommitteesAfterSwapOut+offset > maxCommitteeSize {
		offset = maxCommitteeSize - lenCommitteesAfterSwapOut
	}
	return offset
}

func (s *swapRuleV3) normalSwapOut(committees, substitutes []string,
	lenBeforeSlashedCommittees, lenSlashedCommittees, maxSwapOutPercent, numberOfFixedValidators, minCommitteeSize int,
) ([]string, []string) {
	resNormalSwapOut := []string{}
	resCommittees := []string{}
	normalSwapOutOffset := s.getNormalSwapOutOffset(
		lenBeforeSlashedCommittees, len(substitutes),
		lenSlashedCommittees, maxSwapOutPercent, numberOfFixedValidators,
		minCommitteeSize)

	resCommittees = append(committees[:numberOfFixedValidators], committees[(numberOfFixedValidators+normalSwapOutOffset):]...)
	resNormalSwapOut = committees[numberOfFixedValidators : numberOfFixedValidators+normalSwapOutOffset]

	return resCommittees, resNormalSwapOut
}

func (s *swapRuleV3) getNormalSwapOutOffset(
	lenCommitteesBeforeSlash, lenSubstitutes,
	lenSlashedCommittees, maxSwapOutPercent, numberOfFixedValidators,
	minCommitteeSize int,
) int {
	offset := lenCommitteesBeforeSlash / maxSwapOutPercent
	if lenSlashedCommittees >= offset {
		return 0
	}
	if lenCommitteesBeforeSlash < minCommitteeSize {
		return 0
	}
	if lenSubstitutes == 0 {
		return 0
	}
	offset = offset - lenSlashedCommittees
	if offset > lenSubstitutes {
		offset = lenSubstitutes
	}
	return offset
}

func (s *swapRuleV3) slashingSwapOut(
	committees []string,
	penalty map[string]signaturecounter.Penalty,
	minCommitteeSize, numberOfFixedValidators, maxSlashOutPercent int,
) (
	[]string,
	[]string,
) {
	fixedCommittees := make([]string, len(committees[:numberOfFixedValidators]))
	copy(fixedCommittees, committees[:numberOfFixedValidators])
	flexCommittees := make([]string, len(committees[numberOfFixedValidators:]))
	copy(flexCommittees, committees[numberOfFixedValidators:])
	flexAfterSlashingCommittees := []string{}
	slashingCommittees := []string{}

	slashingOffset := s.getSlashingOffset(len(committees), minCommitteeSize, numberOfFixedValidators, maxSlashOutPercent)
	for _, flexCommittee := range flexCommittees {
		if _, ok := penalty[flexCommittee]; ok && slashingOffset > 0 {
			slashingCommittees = append(slashingCommittees, flexCommittee)
			slashingOffset--
		} else {
			flexAfterSlashingCommittees = append(flexAfterSlashingCommittees, flexCommittee)
		}
	}

	newCommittees := append(fixedCommittees, flexAfterSlashingCommittees...)
	return newCommittees, slashingCommittees
}

func (s *swapRuleV3) getSlashingOffset(
	lenCommittees, minCommitteeSize, numberOfFixedValidators, maxSlashOutPercent int,
) int {
	if lenCommittees == minCommitteeSize {
		return 0
	}
	if lenCommittees == numberOfFixedValidators {
		return 0
	}
	offset := lenCommittees / maxSlashOutPercent
	if numberOfFixedValidators+offset > lenCommittees {
		offset = lenCommittees - numberOfFixedValidators
	}
	return offset
}

func (s *swapRuleV3) clone() SwapRule {
	return &swapRuleV3{}
}

func (s *swapRuleV3) Version() int {
	return swapRuleDCSVersion
}
