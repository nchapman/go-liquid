package liquid

import "testing"

// Upstream parity: unit/resource_limits_unit_test.rb
//
// Ruby's Liquid::ResourceLimits tracks render score, assign score, and
// length, with optional caps. go-liquid has its own resource-limit
// implementation (see `limits` in the evaluator) but does not expose
// the ResourceLimits class API.

func TestUpstream_ResourceLimitsUnit_CumulativeScoresInitializeToZero(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed in go-liquid")
}
func TestUpstream_ResourceLimitsUnit_CumulativeLimitsDefaultToNil(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_CumulativeLimitsConfigurableViaHash(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_CumulativeLimitsConfigurableViaAccessor(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_CumulativeScoresSurviveReset(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_CumulativeScoresAccumulateAcrossResets(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_CumulativeRenderScoreLimitRaises(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_CumulativeAssignScoreLimitRaises(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
func TestUpstream_ResourceLimitsUnit_PerTemplateLimitsStillWorkWithCumulative(t *testing.T) {
	t.Skip("Liquid::ResourceLimits class not exposed")
}
