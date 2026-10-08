// Run through //testdata/notices:program; the compiled program requires this environment value.
fn main() {
    println!("cargo:rustc-env=FIXTURE_BUILD={}", build_only::value());
}
