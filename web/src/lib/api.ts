import type {
  GlobalStats,
  Skill,
  SkillContent,
  SkillDetail,
} from "./types";

const API_URL = process.env.SKILLHUB_API_URL || "https://aithub.space";
const TOKEN = process.env.SKILLHUB_API_TOKEN || "";

const authHeaders: HeadersInit | undefined = TOKEN
  ? { Authorization: `Bearer ${TOKEN}` }
  : undefined;

const EMPTY_STATS: GlobalStats = {
  total_skills: 0,
  total_installs: 0,
  total_contributors: 0,
};

export async function getSkills(): Promise<Skill[]> {
  try {
    const res = await fetch(`${API_URL}/v1/skills`, {
      cache: 'no-store',
      headers: authHeaders,
    });
    if (!res.ok) return [];
    const data = await res.json();
    return (data.skills as Skill[]) || [];
  } catch {
    return [];
  }
}

export async function getGlobalStats(): Promise<GlobalStats> {
  try {
    const res = await fetch(`${API_URL}/v1/stats`, {
      cache: 'no-store',
    });
    if (!res.ok) return EMPTY_STATS;
    return (await res.json()) as GlobalStats;
  } catch {
    return EMPTY_STATS;
  }
}

export async function getSkillDetail(
  namespace: string,
  name: string,
): Promise<SkillDetail | null> {
  try {
    const res = await fetch(
      `${API_URL}/v1/skills/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`,
      { next: { revalidate: 30 }, headers: authHeaders },
    );
    if (!res.ok) return null;
    return (await res.json()) as SkillDetail;
  } catch {
    return null;
  }
}

export async function getSkillContent(
  namespace: string,
  name: string,
  version?: string,
): Promise<SkillContent | null> {
  try {
    const params = version ? `?version=${encodeURIComponent(version)}` : "";
    const res = await fetch(
      `${API_URL}/v1/skills/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/content${params}`,
      { next: { revalidate: 60 }, headers: authHeaders },
    );
    if (!res.ok) return null;
    return (await res.json()) as SkillContent;
  } catch {
    return null;
  }
}
