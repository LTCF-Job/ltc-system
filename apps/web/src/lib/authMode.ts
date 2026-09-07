export interface AuthEnvironment {
  isDev: boolean
  appEnv?: string
  hasSupabaseCredentials: boolean
}

export function isLocalEnvironment(
  env: Pick<AuthEnvironment, 'isDev' | 'appEnv'>
): boolean {
  return env.isDev || env.appEnv === 'local'
}

export function shouldUseSupabaseAuth(env: AuthEnvironment): boolean {
  return !isLocalEnvironment(env) && env.hasSupabaseCredentials
}
