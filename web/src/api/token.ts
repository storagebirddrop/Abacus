// Client-side storage for the optional API bearer token. When the server is
// started with API_TOKEN set, the bundled UI attaches this token to requests.
const KEY = 'abacus-api-token'

export function getToken(): string {
  try {
    return localStorage.getItem(KEY) ?? ''
  } catch {
    return ''
  }
}

export function setToken(token: string): void {
  try {
    if (token) localStorage.setItem(KEY, token)
    else localStorage.removeItem(KEY)
  } catch {
    // ignore storage failures (e.g. private mode)
  }
}
