//! A linked dependency lets the notice fixture distinguish runtime edges from build-only edges.

/// Returns the fixture's linked dependency marker.
///
/// # Examples
///
/// ```
/// assert_eq!(linked::message(), "linked");
/// ```
pub fn message() -> &'static str {
    "linked"
}
