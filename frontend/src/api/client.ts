const API_BASE = (window as any).APP_CONFIG?.apiBase ?? import.meta.env.VITE_API_BASE ?? "";

type RequestOptions = Omit<RequestInit, "body"> & {
  token?: string;
  body?: BodyInit | Record<string, unknown> | null;
};

type ApiErrorResponse = { error?: string; message?: string };

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { token, headers, body, ...rest } = options;

  const response = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers: {
      "Content-Type": "application/json",
      ...(headers ?? {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: body && typeof body !== "string" ? JSON.stringify(body) : (body as BodyInit | undefined)
  });

  if (!response.ok) {
    let message = response.statusText;
    try {
      const data = (await response.json()) as ApiErrorResponse;
      message = data.error ?? data.message ?? message;
    } catch (error) {
      // ignore JSON parse errors
    }
    throw new Error(message || "API request failed");
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export interface LoginResponse {
  tokenType: string;
  accessToken: string;
  accessTokenExpiresAt: string;
  refreshToken: string;
  refreshTokenExpiresAt: string;
  user: { username: string; role: string };
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  return apiFetch<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: { username, password }
  });
}

export async function refresh(refreshToken: string): Promise<LoginResponse> {
  return apiFetch<LoginResponse>("/api/v1/auth/refresh", {
    method: "POST",
    body: { refreshToken }
  });
}

export async function logout(refreshToken: string): Promise<void> {
  await apiFetch<void>("/api/v1/auth/logout", {
    method: "POST",
    body: { refreshToken }
  });
}
