"""Recorded verdict fixtures use the production native compilation path."""

load("@io_bazel_rules_go//go:def.bzl", "go_rule")
load("//supply_chain/private:supply_chain_test.bzl", "REPLAY_ATTRIBUTES", "replay_verdict")

def _impl(ctx):
    report = ctx.actions.declare_file(ctx.label.name + ".verdict/report.txt")
    status = ctx.actions.declare_file(ctx.label.name + ".verdict/status")
    ctx.actions.write(report, ctx.attr.report)
    ctx.actions.write(status, ctx.attr.status)
    return replay_verdict(ctx, report, status)

recorded_verdict_test = go_rule(
    implementation = _impl,
    test = True,
    attrs = REPLAY_ATTRIBUTES | {
        "report": attr.string(mandatory = True),
        "status": attr.string(mandatory = True),
    },
)
