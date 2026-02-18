import type { User } from '../model/types';

export function UserCard({ user }: { user: User }) {
  return (
    <div aria-label="user-card">
      <div>{user.name}</div>
    </div>
  );
}
