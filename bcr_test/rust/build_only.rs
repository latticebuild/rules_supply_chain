//! A build-script dependency contributes no package to the program's notices.

/// Returns a value the build script passes to the fixture program.
///
/// # Examples
///
/// ```
/// assert_eq!(build_only::value(), "built");
/// ```
pub fn value() -> &'static str {
    "built"
}
