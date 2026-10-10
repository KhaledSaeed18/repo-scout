// Applies the last theme before first paint to avoid a light/dark flash.
// Loaded as a blocking file rather than inline so the content security policy
// can forbid inline scripts.
try {
  var t = localStorage.getItem('repo-scout-theme') || 'system'
  var dark = t === 'dark' || (t === 'system' && matchMedia('(prefers-color-scheme: dark)').matches)
  if (dark) document.documentElement.classList.add('dark')
} catch {
  // Storage can be blocked; the default theme then applies.
}
