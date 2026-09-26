package dashboard

import "context"

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Summary(ctx context.Context) (SummaryResponse, error) {
	summary, err := service.repository.Summary(ctx)

	if err != nil {
		return SummaryResponse{}, err
	}

	return toSummaryResponse(summary), nil
}
