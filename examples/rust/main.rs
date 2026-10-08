// Build with bazel build //examples/rust:program.
#[passthrough::keep]
fn main() {
    assert_eq!(linked::message(), "linked");
    assert_eq!(env!("FIXTURE_BUILD"), "built");
}
