import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'
import type { User as UserType } from '@/types/api'

export function useProfile() {
	return useQuery({
		queryKey: ['profile'],
		queryFn: () => api.get<UserType>('/auth/profile'),
		staleTime: 60_000,
	})
}

export function useUpdateProfile() {
	const queryClient = useQueryClient()
	return useMutation({
		mutationFn: (data: { firstName: string; lastName: string; phone: string }) =>
			api.put<UserType>('/users/me', data),
		onSuccess: (updatedUser) => {
			queryClient.setQueryData(['profile'], updatedUser)
			queryClient.invalidateQueries({ queryKey: ['profile'] })
		},
	})
}