Rails.application.routes.draw do
  # Define your application routes per the DSL in https://guides.rubyonrails.org/routing.html

  # Reveal health status on /up that returns 200 if the app boots with no exceptions, otherwise 500.
  # Can be used by load balancers and uptime monitors to verify that the app is live.
  get "up" => "rails/health#show", as: :rails_health_check

  get  "applications",          to: "applications#index"
  post "applications",          to: "applications#create"
  get  "applications/:token",   to: "applications#show"

  # Message search endpoint
  get "applications/:application_token/chats/:chat_number/messages/search",
      to: "messages#search",
      as: :search_messages
end
