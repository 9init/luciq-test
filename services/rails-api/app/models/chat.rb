class Chat < ApplicationRecord
  belongs_to :application, counter_cache: :chats_count
  has_many :messages, dependent: :destroy
  validates :number, presence: true, uniqueness: { scope: :application_id }
  before_validation :set_chat_number, on: :create

  private

  def set_chat_number
    return if number.present?

    # Use Redis or a database sequence in a real scalable setup
    last_number = application.chats.maximum(:number) || 0
    self.number = last_number + 1
  end
end
