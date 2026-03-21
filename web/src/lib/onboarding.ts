export type OnboardingStepKey = 'profile' | 'projects' | 'train';
export type OnboardingStepStatus = 'done' | 'current' | 'next';

export type OnboardingStepState = OnboardingStepStatus;

export interface OnboardingProgressInput {
  hasProfile: boolean;
  hasProjects: boolean;
  hasSession: boolean;
}

export interface OnboardingStepProgress {
  key: OnboardingStepKey;
  status: OnboardingStepStatus;
}

export interface OnboardingProgress {
  completed: boolean;
  currentStep: OnboardingStepKey | null;
  steps: OnboardingStepProgress[];
}

export interface OnboardingSnapshot {
  hasProfile: boolean;
  hasProjects: boolean;
  hasSessions: boolean;
}

export interface OnboardingStep {
  key: OnboardingStepKey;
  state: OnboardingStepState;
}

const STEP_ORDER: OnboardingStepKey[] = ['profile', 'projects', 'train'];

export function buildOnboardingProgress(
  input: OnboardingProgressInput,
): OnboardingProgress {
  const hasProfile = input.hasProfile;
  const hasProjects = input.hasProjects;
  const hasSession = input.hasSession;

  const completedKeys = new Set<OnboardingStepKey>();
  if (hasProfile || hasProjects || hasSession) {
    completedKeys.add('profile');
  }
  if (hasProjects || hasSession) {
    completedKeys.add('projects');
  }
  if (hasSession) {
    completedKeys.add('train');
  }

  const currentStep =
    STEP_ORDER.find((step) => !completedKeys.has(step)) ?? null;

  return {
    completed: currentStep === null,
    currentStep,
    steps: STEP_ORDER.map((key) => ({
      key,
      status: completedKeys.has(key)
        ? 'done'
        : key === currentStep
          ? 'current'
          : 'next',
    })),
  };
}

export function buildOnboardingHref(step: OnboardingStepKey): string {
  return buildOnboardingTarget(step);
}

export function buildOnboardingSnapshot(
  dashboard:
    | {
        profile?: unknown;
        current_session?: unknown;
        recent_sessions?: unknown[];
      }
    | null
    | undefined,
  projects: unknown[] | null | undefined,
): OnboardingSnapshot {
  return {
    hasProfile: Boolean(dashboard?.profile),
    hasProjects: Boolean(projects?.length),
    hasSessions: Boolean(
      dashboard?.current_session ||
      (dashboard?.recent_sessions?.length ?? 0) > 0,
    ),
  };
}

export function shouldShowOnboarding(snapshot: OnboardingSnapshot): boolean {
  return !snapshot.hasSessions;
}

export function resolveOnboardingSteps(
  snapshot: OnboardingSnapshot,
): OnboardingStep[] {
  return buildOnboardingProgress({
    hasProfile: snapshot.hasProfile,
    hasProjects: snapshot.hasProjects,
    hasSession: snapshot.hasSessions,
  }).steps.map((step) => ({
    key: step.key,
    state: step.status,
  }));
}

export function buildOnboardingTarget(
  step: OnboardingStepKey,
  options?: { skipProjects?: boolean },
): string {
  const base =
    step === 'profile'
      ? '/profile'
      : step === 'projects'
        ? '/projects'
        : '/train';
  const query = new URLSearchParams({ onboarding: '1' });
  if (options?.skipProjects) {
    query.set('skip_projects', '1');
  }
  return `${base}?${query.toString()}`;
}
