class ApplicationsController < ApplicationController
  def index
    page = params.fetch(:page, 1).to_i
    page = 1 if page < 1
    per_page = params.fetch(:per_page, 20).to_i
    per_page = 1 if per_page < 1

    total = Application.count
    applications = Application.order(created_at: :desc)
                              .offset((page - 1) * per_page)
                              .limit(per_page)

    render json: {
      data: applications,
      meta: {
        page: page,
        per_page: per_page,
        total: total,
        total_pages: (total.to_f / per_page).ceil
      }
    }
  end

  def show
    token = params[:token]
    cache_key = "application:token:#{token}"

    app = $redis.get(cache_key)

    if app
      render json: JSON.parse(app)
      return
    else
      application = Application.find_by!(token: token)
      $redis.set(cache_key, application.to_json, ex: 30 * 60) # expire in 30 minutes
    end

    render json: application
  end

  def create
    application = Application.create!(application_params)
    render json: { token: application.token, name: application.name }
  end

  private

  def application_params
    params.require(:application).permit(:name)
  end
end
