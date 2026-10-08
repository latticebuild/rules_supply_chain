// Build with bazel build //testdata/notices:program.
#[passthrough::keep]
fn main() {
    assert_eq!(linked::message(), "linked");
    assert_eq!(env!("FIXTURE_BUILD"), "built");
}
