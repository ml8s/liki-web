// Preserve the free-chart draft and hand off to onboarding.
const form = document.querySelector<HTMLFormElement>('#paipanHook');

form?.addEventListener('submit', (event) => {
  event.preventDefault();
  const values = new FormData(form);
  const birthDate = values.get('date');
  const birthTime = values.get('time');
  const gender = values.get('gender');
  if (typeof birthDate !== 'string' || typeof birthTime !== 'string' || typeof gender !== 'string') return;
  if (!birthDate || !birthTime || !gender) return;

  try {
    sessionStorage.setItem('liki-draft', JSON.stringify({
      birth_date: birthDate,
      birth_time: birthTime,
      gender,
      calendar: 'solar',
    }));
  } catch {
    // Private browsing can reject storage; still continue to onboarding.
  }

  location.href = '/app/birth-profiles/new';
});
