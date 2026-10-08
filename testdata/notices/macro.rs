//! The fixture uses an attribute macro so the notice aspect must visit proc_macro_deps.

use proc_macro::TokenStream;

/// Returns the annotated item unchanged, accepting any attribute tokens.
///
/// # Examples
///
/// ```
/// #[passthrough::keep]
/// fn value() -> u32 { 3 }
/// assert_eq!(value(), 3);
/// ```
#[proc_macro_attribute]
pub fn keep(_attributes: TokenStream, item: TokenStream) -> TokenStream {
    item
}
