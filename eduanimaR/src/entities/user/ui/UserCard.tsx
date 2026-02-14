import type { User } from '../model/types';

export function UserCard({ user }: { user: User }) {
  return (
    <div>
      <div>User</div>
      <div>{user.name}</div>
    </div>
  );
}
