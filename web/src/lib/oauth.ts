import { API_BASE_URL } from '@/lib/api';

export interface OAuthProvider {
  id: 'github' | 'google' | 'oidc' | 'lark' | 'wechat' | 'telegram';
  label: string;
  loginPath: string;
  bindPath?: string;
}

export const oauthProviders: OAuthProvider[] = [
  { id: 'github', label: 'GitHub', loginPath: '/oauth/github', bindPath: '/oauth/github/bind' },
  { id: 'google', label: 'Google', loginPath: '/oauth/google' },
  { id: 'oidc', label: 'OIDC', loginPath: '/oauth/oidc', bindPath: '/oauth/oidc/bind' },
  { id: 'lark', label: '飞书', loginPath: '/oauth/lark', bindPath: '/oauth/lark/bind' },
  { id: 'wechat', label: '微信', loginPath: '/oauth/wechat', bindPath: '/oauth/wechat/bind' },
  { id: 'telegram', label: 'Telegram', loginPath: '/oauth/telegram/login', bindPath: '/oauth/telegram/bind' },
];

export const bindableOAuthProviders = oauthProviders.filter((provider) => provider.bindPath);

export function getApiRedirectURL(path: string) {
  const base = API_BASE_URL.replace(/\/$/, '');
  const nextPath = path.startsWith('/') ? path : `/${path}`;
  return `${base}${nextPath}`;
}

export function redirectToURL(url: string) {
  window.location.assign(url);
}

export function redirectToApiPath(path: string) {
  redirectToURL(getApiRedirectURL(path));
}
